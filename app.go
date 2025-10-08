/**
 * Shotgun Code - Main Application
 *
 * This is the core backend for Shotgun Code, a desktop application built with Wails (Go + Vue.js)
 * that generates massive codebase snapshots for AI coding assistants.
 *
 * Key Features:
 * - Unlimited context generation (no size limits)
 * - File tree traversal with gitignore and custom ignore support
 * - Background job queue for async operations
 * - File watching for real-time updates
 * - LLM integration for direct API calls
 * - Clipboard management with multi-tier fallback
 *
 * Architecture:
 * - App: Main application struct, coordinates all components
 * - ContextGenerator: Handles context generation and file tree building
 * - JobQueue: Manages background jobs with progress tracking
 * - Watchman: File system watcher for real-time updates
 * - LLMClient: Unified client for multiple LLM providers
 */
package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/adrg/xdg"                        // XDG Base Directory Specification for config files
	"github.com/fsnotify/fsnotify"               // File system notifications
	gitignore "github.com/sabhiram/go-gitignore" // Gitignore pattern matching
	"github.com/wailsapp/wails/v2/pkg/runtime"   // Wails runtime for logging and dialogs
)

// No size limit - removed to allow unlimited context generation
// Previous versions had a 10MB limit, but this has been removed to support
// generating context for very large codebases without restrictions

//go:embed ignore.glob
var defaultCustomIgnoreRulesContent string // Default custom ignore patterns embedded at compile time

// defaultCustomPromptRulesContent is the default value for custom prompt rules
// when no user-defined rules are set
const defaultCustomPromptRulesContent = "no additional rules"

// AppSettings represents the persistent application settings
// These are stored in the user's config directory (XDG_CONFIG_HOME/shotgun-code/settings.json)
type AppSettings struct {
	CustomIgnoreRules string `json:"customIgnoreRules"` // User-defined file ignore patterns (glob format)
	CustomPromptRules string `json:"customPromptRules"` // User-defined prompt customization rules
}

// App is the main application struct that coordinates all components
// It serves as the central hub for the Wails application
type App struct {
	ctx                         context.Context      // Application context for lifecycle management
	contextGenerator            *ContextGenerator    // Handles context generation operations
	fileWatcher                 *Watchman            // File system watcher for real-time updates
	jobQueue                    *JobQueue            // Background job queue for async operations
	settings                    AppSettings          // User settings loaded from config file
	currentCustomIgnorePatterns *gitignore.GitIgnore // Compiled custom ignore patterns
	configPath                  string               // Path to the settings.json config file
	useGitignore                bool                 // Whether to respect .gitignore files
	useCustomIgnore             bool                 // Whether to apply custom ignore patterns
	projectGitignore            *gitignore.GitIgnore // Compiled .gitignore for the current project
}

// NewApp creates a new App instance
// This is called by Wails during application initialization
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the application starts
// It initializes all components and loads user settings
//
// Parameters:
//   - ctx: Application context for lifecycle management
func (a *App) startup(ctx context.Context) {
	// Store the application context for use throughout the app
	a.ctx = ctx

	// Initialize core components
	a.contextGenerator = NewContextGenerator(a) // Handles context generation
	a.fileWatcher = NewWatchman(a)              // Watches for file system changes
	a.jobQueue = NewJobQueue(a)                 // Manages background jobs

	// Set default ignore behavior (can be toggled by user in UI)
	a.useGitignore = true    // Respect .gitignore files by default
	a.useCustomIgnore = true // Apply custom ignore patterns by default

	// Get the path to the user's config file using XDG Base Directory spec
	// On Linux: ~/.config/shotgun-code/settings.json
	// On macOS: ~/Library/Application Support/shotgun-code/settings.json
	// On Windows: %APPDATA%\shotgun-code\settings.json
	configFilePath, err := xdg.ConfigFile("shotgun-code/settings.json")
	if err != nil {
		runtime.LogErrorf(a.ctx, "Error getting config file path: %v. Using defaults and will attempt to save later if rules are modified.", err)
		// If we can't get the config path, we'll use defaults
		// Settings will still work in-memory, but won't persist across restarts
	}
	a.configPath = configFilePath

	// Load user settings from disk (or use defaults if file doesn't exist)
	a.loadSettings()

	// Ensure CustomPromptRules has a default value if it's empty after loading
	// This prevents the UI from showing an empty state
	if strings.TrimSpace(a.settings.CustomPromptRules) == "" {
		a.settings.CustomPromptRules = defaultCustomPromptRulesContent
	}

	// Start a background goroutine for periodic cleanup of old jobs
	// This prevents the job queue from growing indefinitely
	// Runs every 5 minutes and removes jobs older than 1 hour
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Clean up jobs older than 1 hour
				a.jobQueue.CleanupOldJobs(1 * time.Hour)
			case <-ctx.Done():
				// Application is shutting down, exit the cleanup goroutine
				return
			}
		}
	}()
}

// FileNode represents a file or directory in the file tree
// This structure is sent to the frontend for display in the file selection UI
type FileNode struct {
	Name            string      `json:"name"`               // File or directory name (without path)
	Path            string      `json:"path"`               // Full absolute path on the file system
	RelPath         string      `json:"relPath"`            // Path relative to the selected project root
	IsDir           bool        `json:"isDir"`              // True if this is a directory, false if it's a file
	Children        []*FileNode `json:"children,omitempty"` // Child nodes (only for directories)
	IsGitignored    bool        `json:"isGitignored"`       // True if this path matches a .gitignore rule
	IsCustomIgnored bool        `json:"isCustomIgnored"`    // True if this path matches a custom ignore pattern
	Size            int64       `json:"size"`               // File size in bytes (0 for directories)
	IsBinary        bool        `json:"isBinary"`           // True if this is a binary file (detected by content analysis)
}

// FileContentResult represents the result of reading a file's content
// Used by ReadFileContents to return file data with validation status
type FileContentResult struct {
	Path     string `json:"path"`     // Relative path of the file
	Content  string `json:"content"`  // File content (empty if binary or error)
	Size     int64  `json:"size"`     // File size in bytes
	IsBinary bool   `json:"isBinary"` // True if file is binary
	Error    string `json:"error"`    // Error message if read failed (empty on success)
}

// ============================================================================
// Binary File Detection Utilities
// ============================================================================

// Common binary file extensions that should always be treated as binary
// This list is used as a fast-path check before content analysis
var binaryExtensions = map[string]bool{
	// Executables and libraries
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".o": true, ".a": true, ".lib": true, ".obj": true, ".class": true,
	".pyc": true, ".pyo": true, ".elc": true,

	// Archives and compressed files
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
	".7z": true, ".rar": true, ".tgz": true, ".tbz2": true, ".lz": true,
	".lzma": true, ".z": true, ".cab": true, ".iso": true, ".dmg": true,
	".pkg": true, ".deb": true, ".rpm": true,

	// Images
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
	".tiff": true, ".tif": true, ".webp": true, ".ico": true, ".icns": true,
	".psd": true, ".ai": true, ".eps": true, ".svg": true, ".raw": true,
	".cr2": true, ".nef": true, ".orf": true, ".sr2": true,

	// Audio
	".mp3": true, ".wav": true, ".ogg": true, ".flac": true, ".aac": true,
	".wma": true, ".m4a": true, ".opus": true, ".ape": true, ".alac": true,

	// Video
	".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
	".mkv": true, ".webm": true, ".m4v": true, ".mpg": true, ".mpeg": true,
	".3gp": true, ".ogv": true,

	// Documents (binary formats)
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,
	".pages": true, ".numbers": true, ".key": true,

	// Fonts
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,

	// Databases
	".db": true, ".sqlite": true, ".sqlite3": true, ".mdb": true, ".accdb": true,

	// Other binary formats
	".swf": true, ".fla": true, ".jar": true, ".war": true, ".ear": true,
	".apk": true, ".ipa": true, ".wasm": true,
}

// Binary filenames (files without extensions that should be treated as binary)
// These are typically OS-specific metadata or cache files
var binaryFilenames = map[string]bool{
	".DS_Store":   true, // macOS folder metadata
	"Thumbs.db":   true, // Windows thumbnail cache
	"desktop.ini": true, // Windows folder settings
}

// isBinaryFile determines if a file is binary by checking extension and content
// This function uses multiple detection strategies for accuracy:
// 1. Extension-based detection (fast path)
// 2. Null byte detection (reliable for most binary files)
// 3. UTF-8 validation (text files should be valid UTF-8)
// 4. Non-printable character ratio analysis
//
// Parameters:
//   - filePath: Path to the file to check
//
// Returns:
//   - bool: true if file is binary, false if text
//   - error: Error if file cannot be read
func isBinaryFile(filePath string) (bool, error) {
	// Validate input
	if filePath == "" {
		return false, fmt.Errorf("file path is empty")
	}

	// Check filename first (for files without extensions like .DS_Store)
	filename := filepath.Base(filePath)
	if binaryFilenames[filename] {
		return true, nil
	}

	// Check extension (fast path)
	ext := strings.ToLower(filepath.Ext(filePath))
	if binaryExtensions[ext] {
		return true, nil
	}

	// Open file for content analysis
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 8KB for analysis (sufficient for most files)
	// Reading more would be wasteful for large files
	const sampleSize = 8192
	buffer := make([]byte, sampleSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Empty files are considered text
	if n == 0 {
		return false, nil
	}

	// Trim buffer to actual bytes read
	buffer = buffer[:n]

	// Strategy 1: Check for null bytes (strong indicator of binary)
	if bytes.Contains(buffer, []byte{0}) {
		return true, nil
	}

	// Strategy 2: Validate UTF-8 encoding
	// Text files should be valid UTF-8 (or ASCII, which is valid UTF-8)
	if !utf8.Valid(buffer) {
		return true, nil
	}

	// Strategy 3: Analyze non-printable character ratio
	// Text files should have mostly printable characters
	nonPrintable := 0
	for _, b := range buffer {
		// Count non-printable characters (excluding common whitespace)
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonPrintable++
		}
		// High-value bytes (127-255) that aren't valid UTF-8 continuation bytes
		if b == 127 || (b >= 128 && b < 192) {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider it binary
	// This threshold is conservative to avoid false positives
	threshold := float64(len(buffer)) * 0.30
	if float64(nonPrintable) > threshold {
		return true, nil
	}

	// Passed all checks - consider it a text file
	return false, nil
}

// SelectDirectory opens a dialog to select a directory and returns the chosen path
//
// This method opens a native directory picker dialog and returns the selected path.
// It's used by the frontend to allow users to choose their project folder.
//
// Returns:
//   - string: The selected directory path, or empty string if cancelled
//   - error: Error if dialog fails to open
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Project Folder",
	})
}

// ReadFileContents reads the contents of multiple files with validation
// This method is used by the frontend to read file contents for copying to clipboard
// It includes binary detection, size validation, and proper error handling
//
// Parameters:
//   - rootDir: Root directory path (for resolving relative paths)
//   - relativePaths: Array of relative file paths to read
//
// Returns:
//   - []FileContentResult: Array of results with content, size, and error info
//   - error: Error if rootDir is invalid or operation fails
func (a *App) ReadFileContents(rootDir string, relativePaths []string) ([]FileContentResult, error) {
	// Validate inputs
	if rootDir == "" {
		return nil, fmt.Errorf("root directory is empty")
	}

	if relativePaths == nil {
		return nil, fmt.Errorf("relative paths array is nil")
	}

	// Check if root directory exists
	rootInfo, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("root directory does not exist: %w", err)
	}

	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", rootDir)
	}

	runtime.LogInfof(a.ctx, "ReadFileContents: Reading %d files from %s", len(relativePaths), rootDir)

	// Prepare results array
	results := make([]FileContentResult, 0, len(relativePaths))

	// Process each file
	for _, relPath := range relativePaths {
		result := FileContentResult{
			Path: relPath,
		}

		// Validate relative path
		if relPath == "" {
			result.Error = "empty file path"
			results = append(results, result)
			continue
		}

		// Construct absolute path
		absPath := filepath.Join(rootDir, relPath)

		// Security check: ensure path is within root directory
		cleanPath := filepath.Clean(absPath)
		cleanRoot := filepath.Clean(rootDir)
		if !strings.HasPrefix(cleanPath, cleanRoot) {
			result.Error = "path is outside root directory (security violation)"
			runtime.LogWarningf(a.ctx, "Security violation: attempted to read %s outside root %s", cleanPath, cleanRoot)
			results = append(results, result)
			continue
		}

		// Check if file exists
		fileInfo, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				result.Error = "file not found"
			} else {
				result.Error = fmt.Sprintf("stat error: %v", err)
			}
			results = append(results, result)
			continue
		}

		// Skip directories
		if fileInfo.IsDir() {
			result.Error = "path is a directory, not a file"
			results = append(results, result)
			continue
		}

		// Get file size
		result.Size = fileInfo.Size()

		// Check for excessively large files (>100MB warning threshold)
		const maxRecommendedSize = 100 * 1024 * 1024 // 100MB
		if result.Size > maxRecommendedSize {
			runtime.LogWarningf(a.ctx, "Large file detected: %s (%d bytes)", relPath, result.Size)
		}

		// Detect if file is binary
		isBinary, err := isBinaryFile(absPath)
		if err != nil {
			result.Error = fmt.Sprintf("binary detection failed: %v", err)
			results = append(results, result)
			continue
		}

		result.IsBinary = isBinary

		// Skip reading content for binary files
		if isBinary {
			result.Content = ""
			runtime.LogDebugf(a.ctx, "Skipping binary file: %s", relPath)
			results = append(results, result)
			continue
		}

		// Read file content
		content, err := os.ReadFile(absPath)
		if err != nil {
			result.Error = fmt.Sprintf("read error: %v", err)
			results = append(results, result)
			continue
		}

		// Validate UTF-8 encoding
		if !utf8.Valid(content) {
			result.Error = "file contains invalid UTF-8 (possibly binary)"
			result.IsBinary = true
			runtime.LogWarningf(a.ctx, "Invalid UTF-8 in file: %s", relPath)
			results = append(results, result)
			continue
		}

		// Success - store content
		result.Content = string(content)
		results = append(results, result)
	}

	runtime.LogInfof(a.ctx, "ReadFileContents: Successfully processed %d files", len(results))
	return results, nil
}

// ListFiles lists files and folders in a directory, parsing .gitignore if present
func (a *App) ListFiles(dirPath string) ([]*FileNode, error) {
	runtime.LogDebugf(a.ctx, "ListFiles called for directory: %s", dirPath)

	a.projectGitignore = nil        // Reset for the new directory
	var gitIgn *gitignore.GitIgnore // For .gitignore in the project directory
	gitignorePath := filepath.Join(dirPath, ".gitignore")
	runtime.LogDebugf(a.ctx, "Attempting to find .gitignore at: %s", gitignorePath)
	if _, err := os.Stat(gitignorePath); err == nil {
		runtime.LogDebugf(a.ctx, ".gitignore found at: %s", gitignorePath)
		gitIgn, err = gitignore.CompileIgnoreFile(gitignorePath)
		if err != nil {
			runtime.LogWarningf(a.ctx, "Error compiling .gitignore file at %s: %v", gitignorePath, err)
			gitIgn = nil
		} else {
			a.projectGitignore = gitIgn // Store the compiled project-specific gitignore
			runtime.LogDebug(a.ctx, ".gitignore compiled successfully.")
		}
	} else {
		runtime.LogDebugf(a.ctx, ".gitignore not found at %s (os.Stat error: %v)", gitignorePath, err)
		gitIgn = nil
	}

	// App-level custom ignore patterns are in a.currentCustomIgnorePatterns

	rootNode := &FileNode{
		Name:         filepath.Base(dirPath),
		Path:         dirPath,
		RelPath:      ".",
		IsDir:        true,
		IsGitignored: false, // Root itself is not gitignored by default
		// IsCustomIgnored for root is also false by default, specific patterns would be needed
		IsCustomIgnored: a.currentCustomIgnorePatterns != nil && a.currentCustomIgnorePatterns.MatchesPath("."),
	}

	// No timeout for tree building - allow unlimited time for huge codebases
	// Previous 30-second timeout was causing failures on large projects
	ctx := a.ctx

	children, err := buildTreeRecursive(ctx, dirPath, dirPath, gitIgn, a.currentCustomIgnorePatterns, 0)
	if err != nil {
		return []*FileNode{rootNode}, fmt.Errorf("error building children tree for %s: %w", dirPath, err)
	}
	rootNode.Children = children

	return []*FileNode{rootNode}, nil
}

func buildTreeRecursive(ctx context.Context, currentPath, rootPath string, gitIgn *gitignore.GitIgnore, customIgn *gitignore.GitIgnore, depth int) ([]*FileNode, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}

	var nodes []*FileNode
	for _, entry := range entries {
		nodePath := filepath.Join(currentPath, entry.Name())
		relPath, _ := filepath.Rel(rootPath, nodePath)
		// For gitignore matching, paths should generally be relative to the .gitignore file (rootPath)
		// and use OS-specific separators. go-gitignore handles this.

		isGitignored := false
		isCustomIgnored := false
		pathToMatch := relPath
		if entry.IsDir() {
			if !strings.HasSuffix(pathToMatch, string(os.PathSeparator)) {
				pathToMatch += string(os.PathSeparator)
			}
		}

		if gitIgn != nil {
			isGitignored = gitIgn.MatchesPath(pathToMatch)
		}
		if customIgn != nil {
			isCustomIgnored = customIgn.MatchesPath(pathToMatch)
		}

		if depth < 2 || strings.Contains(relPath, "node_modules") || strings.HasSuffix(relPath, ".log") {
			fmt.Printf("Checking path: '%s' (original relPath: '%s'), IsDir: %v, Gitignored: %v, CustomIgnored: %v\n", pathToMatch, relPath, entry.IsDir(), isGitignored, isCustomIgnored)
		}

		// Initialize node with basic information
		node := &FileNode{
			Name:            entry.Name(),
			Path:            nodePath,
			RelPath:         relPath,
			IsDir:           entry.IsDir(),
			IsGitignored:    isGitignored,
			IsCustomIgnored: isCustomIgnored,
			Size:            0,
			IsBinary:        false,
		}

		if entry.IsDir() {
			// If it's a directory, recursively call buildTree
			// Only recurse if not ignored
			if !isGitignored && !isCustomIgnored {
				children, err := buildTreeRecursive(ctx, nodePath, rootPath, gitIgn, customIgn, depth+1)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return nil, err // Propagate cancellation
					}
					// runtime.LogWarnf(ctx, "Error building subtree for %s: %v", nodePath, err) // Use ctx if available
					runtime.LogWarningf(context.Background(), "Error building subtree for %s: %v", nodePath, err) // Fallback for now
					// Decide: skip this dir or return error up. For now, skip with log.
				} else {
					node.Children = children
				}
			}
			// Directory size remains 0
		} else {
			// For files, get size and detect if binary
			fileInfo, err := entry.Info()
			if err != nil {
				runtime.LogWarningf(context.Background(), "Error getting file info for %s: %v", nodePath, err)
			} else {
				node.Size = fileInfo.Size()

				// Detect if file is binary (only if not already ignored)
				// Skip binary detection for ignored files to save time
				if !isGitignored && !isCustomIgnored {
					isBinary, err := isBinaryFile(nodePath)
					if err != nil {
						runtime.LogWarningf(context.Background(), "Error detecting binary for %s: %v", nodePath, err)
						// On error, assume it's binary to be safe
						node.IsBinary = true
					} else {
						node.IsBinary = isBinary
					}
				}
			}
		}
		nodes = append(nodes, node)
	}
	// Sort nodes: directories first, then files, then alphabetically
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].IsDir && !nodes[j].IsDir {
			return true
		}
		if !nodes[i].IsDir && nodes[j].IsDir {
			return false
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	return nodes, nil
}

// ContextGenerator manages the asynchronous generation of shotgun context
// It handles background generation with cancellation support and progress tracking
//
// Key Features:
// - Asynchronous generation in background goroutines
// - Cancellation support (can cancel ongoing generation)
// - Progress tracking via events emitted to frontend
// - Thread-safe with mutex protection
// - No size limits (unlimited context generation)
type ContextGenerator struct {
	app                *App               // Reference to main app for accessing Wails runtime
	mu                 sync.Mutex         // Protects concurrent access to cancel func and token
	currentCancelFunc  context.CancelFunc // Function to cancel the current generation job
	currentCancelToken interface{}        // Unique token to identify the current job (prevents race conditions)
}

// NewContextGenerator creates a new ContextGenerator instance
//
// Parameters:
//   - app: Reference to the main App for accessing Wails runtime
//
// Returns:
//   - *ContextGenerator: New context generator instance
func NewContextGenerator(app *App) *ContextGenerator {
	return &ContextGenerator{app: app}
}

// requestShotgunContextGenerationInternal starts a new context generation job
// If a previous job is running, it will be cancelled first
//
// This is an internal method called by the App's public wrapper method
// It runs the generation in a background goroutine and emits progress events
//
// Parameters:
//   - rootDir: Root directory to generate context from
//   - excludedPaths: List of paths to exclude from the context
func (cg *ContextGenerator) requestShotgunContextGenerationInternal(rootDir string, excludedPaths []string) {
	cg.mu.Lock()

	// Cancel any previous generation job that might still be running
	if cg.currentCancelFunc != nil {
		runtime.LogDebug(cg.app.ctx, "Cancelling previous context generation job.")
		cg.currentCancelFunc()
	}

	// Create a new context with cancellation support for this generation job
	genCtx, cancel := context.WithCancel(cg.app.ctx)

	// Create a unique token to identify this specific job
	// This prevents race conditions where a new job might clear the cancel func of another job
	myToken := new(struct{})
	cg.currentCancelFunc = cancel
	cg.currentCancelToken = myToken

	// Log the start of generation (no size limit)
	runtime.LogInfof(cg.app.ctx, "Starting new shotgun context generation for: %s (no size limit).", rootDir)
	cg.mu.Unlock()

	go func(tokenForThisJob interface{}) {
		jobStartTime := time.Now()
		defer func() {
			cg.mu.Lock()
			if cg.currentCancelToken == tokenForThisJob { // Only clear if it's still this job's token
				cg.currentCancelFunc = nil
				cg.currentCancelToken = nil
				runtime.LogDebug(cg.app.ctx, "Cleared currentCancelFunc for completed/cancelled job (token match).")
			} else {
				runtime.LogDebug(cg.app.ctx, "currentCancelFunc was replaced by a newer job (token mismatch); not clearing.")
			}
			cg.mu.Unlock()
			runtime.LogInfof(cg.app.ctx, "Shotgun context generation goroutine finished in %s", time.Since(jobStartTime))
		}()

		if genCtx.Err() != nil { // Check for immediate cancellation
			runtime.LogInfo(cg.app.ctx, fmt.Sprintf("Context generation for %s cancelled before starting: %v", rootDir, genCtx.Err()))
			return
		}

		output, err := cg.app.generateShotgunOutputWithProgress(genCtx, rootDir, excludedPaths)

		select {
		case <-genCtx.Done():
			errMsg := fmt.Sprintf("Shotgun context generation cancelled for %s: %v", rootDir, genCtx.Err())
			runtime.LogInfo(cg.app.ctx, errMsg) // Changed from LogWarn
			runtime.EventsEmit(cg.app.ctx, "shotgunContextError", errMsg)
		default:
			if err != nil {
				errMsg := fmt.Sprintf("Error generating shotgun output for %s: %v", rootDir, err)
				runtime.LogError(cg.app.ctx, errMsg)
				runtime.EventsEmit(cg.app.ctx, "shotgunContextError", errMsg)
			} else {
				// Context generation successful - no size limit enforced
				finalSize := len(output)
				successMsg := fmt.Sprintf("Shotgun context generated successfully for %s. Size: %d bytes.", rootDir, finalSize)
				runtime.LogInfo(cg.app.ctx, successMsg)
				runtime.EventsEmit(cg.app.ctx, "shotgunContextGenerated", output)
			}
		}
	}(myToken) // Pass the token to the goroutine
}

// RequestShotgunContextGeneration is the method bound to Wails.
func (a *App) RequestShotgunContextGeneration(rootDir string, excludedPaths []string) {
	// Validate context generator
	if a.contextGenerator == nil {
		// This should not happen if startup initializes it correctly
		runtime.LogError(a.ctx, "ContextGenerator not initialized")
		runtime.EventsEmit(a.ctx, "shotgunContextError", "Internal error: ContextGenerator not initialized")
		return
	}

	// Validate root directory
	if strings.TrimSpace(rootDir) == "" {
		runtime.LogError(a.ctx, "RequestShotgunContextGeneration called with empty rootDir")
		runtime.EventsEmit(a.ctx, "shotgunContextError", "No project folder specified")
		return
	}

	// Check if directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		runtime.LogErrorf(a.ctx, "RequestShotgunContextGeneration: directory does not exist: %s", rootDir)
		runtime.EventsEmit(a.ctx, "shotgunContextError", fmt.Sprintf("Directory does not exist: %s", rootDir))
		return
	}

	// Validate excludedPaths (ensure it's not nil)
	if excludedPaths == nil {
		excludedPaths = []string{}
	}

	a.contextGenerator.requestShotgunContextGenerationInternal(rootDir, excludedPaths)
}

// CancelShotgunContextGeneration cancels the currently running context generation
// This method is exposed to the frontend via Wails binding
//
// Returns:
//   - error: Error if no generation is running or cancellation fails
func (a *App) CancelShotgunContextGeneration() error {
	// Validate context generator
	if a.contextGenerator == nil {
		return fmt.Errorf("context generator not initialized")
	}

	a.contextGenerator.mu.Lock()
	defer a.contextGenerator.mu.Unlock()

	// Check if there's a running generation to cancel
	if a.contextGenerator.currentCancelFunc == nil {
		return fmt.Errorf("no context generation is currently running")
	}

	// Cancel the generation
	runtime.LogInfo(a.ctx, "Cancelling shotgun context generation by user request")
	a.contextGenerator.currentCancelFunc()

	// Clear the cancel function and token
	a.contextGenerator.currentCancelFunc = nil
	a.contextGenerator.currentCancelToken = nil

	return nil
}

// ============================================================================
// Job Queue Methods (Wails-bound)
// ============================================================================

// CancelJob cancels a running job by its ID
// This method is exposed to the frontend via Wails binding
//
// Parameters:
//   - jobID: Unique identifier of the job to cancel
//
// Returns:
//   - error: Error if job not found or cannot be cancelled
func (a *App) CancelJob(jobID string) error {
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}
	return a.jobQueue.CancelJob(jobID)
}

// GetJobStatuses returns the current status of all jobs
// This method is exposed to the frontend via Wails binding
//
// Returns:
//   - []Job: List of all jobs with their current status
func (a *App) GetJobStatuses() []Job {
	if a.jobQueue == nil {
		return []Job{}
	}
	return a.jobQueue.GetJobStatuses()
}

// ============================================================================
// LLM Integration Methods (Wails-bound)
// ============================================================================

// CallLLMAPI calls an LLM API (Google AI Studio, OpenAI, or Anthropic)
// This method runs the LLM call as a background job and returns the job ID
//
// Parameters:
//   - provider: LLM provider (google, openai, anthropic)
//   - apiKey: API key for the provider
//   - prompt: The prompt to send to the LLM
//   - model: Model name (e.g., gemini-1.5-pro, gpt-4, claude-3-5-sonnet-20241022)
//   - temperature: Temperature for generation (0.0-1.0)
//   - maxTokens: Maximum tokens to generate
//
// Returns:
//   - string: Job ID for tracking the LLM call
//   - error: Error if job creation fails
func (a *App) CallLLMAPI(provider, apiKey, prompt, model string, temperature float64, maxTokens int) (string, error) {
	if a.jobQueue == nil {
		return "", fmt.Errorf("job queue not initialized")
	}

	// Create LLM client
	client := NewLLMClient(a)

	// Add LLM call as a background job
	jobID := a.jobQueue.AddJob("llm_call", func(ctx context.Context) error {
		// Create LLM request
		req := LLMRequest{
			Provider:    provider,
			APIKey:      apiKey,
			Prompt:      prompt,
			Model:       model,
			Temperature: temperature,
			MaxTokens:   maxTokens,
		}

		// Call LLM API
		resp, err := client.CallLLM(ctx, req)
		if err != nil {
			return err
		}

		// Emit response to frontend
		runtime.EventsEmit(a.ctx, "llmResponseReceived", resp)
		return nil
	})

	return jobID, nil
}

// GeneratePrompt generates a complete prompt from context, mode, and task description
//
// This method combines the generated context with the user's task description and mode
// to create a complete prompt ready for LLM execution.
//
// Parameters:
//   - context: The generated codebase context (from shotgun generation)
//   - mode: The selected mode (dev, architect, debug, tasks)
//   - taskDescription: User's description of what they want to accomplish
//   - customRules: Optional custom rules/constraints
//
// Returns:
//   - string: The complete formatted prompt
func (a *App) GeneratePrompt(context, mode, taskDescription, customRules string) string {
	// Validate inputs
	if strings.TrimSpace(context) == "" {
		runtime.LogWarning(a.ctx, "GeneratePrompt called with empty context")
		context = "[No codebase context available]"
	}

	if strings.TrimSpace(mode) == "" {
		runtime.LogWarning(a.ctx, "GeneratePrompt called with empty mode, defaulting to 'dev'")
		mode = "dev"
	}

	if strings.TrimSpace(taskDescription) == "" {
		runtime.LogWarning(a.ctx, "GeneratePrompt called with empty task description")
		taskDescription = "[No task description provided]"
	}

	var modeInstructions string

	switch mode {
	case "dev":
		modeInstructions = `You are an expert software developer. Your task is to generate code changes based on the user's request.
- Provide complete, working code
- Follow the existing code style and patterns
- Include necessary imports and dependencies
- Generate a git diff format output that can be applied directly`

	case "architect":
		modeInstructions = `You are a software architect. Your task is to design system architecture and plan refactoring.
- Provide high-level architectural decisions
- Explain design patterns and trade-offs
- Create clear diagrams or descriptions
- Suggest implementation steps`

	case "debug":
		modeInstructions = `You are a debugging expert. Your task is to identify and fix bugs, security issues, and code smells.
- Analyze the code for potential issues
- Identify security vulnerabilities
- Suggest fixes with explanations
- Provide git diff format for fixes`

	case "tasks":
		modeInstructions = `You are a project manager. Your task is to generate task lists and update documentation.
- Break down work into actionable tasks
- Estimate complexity and dependencies
- Update relevant documentation
- Provide clear acceptance criteria`

	default:
		runtime.LogWarningf(a.ctx, "Unknown mode '%s', using default instructions", mode)
		modeInstructions = "You are an AI assistant helping with software development tasks."
	}

	prompt := fmt.Sprintf(`%s

# Codebase Context

%s

# Task

%s`, modeInstructions, context, taskDescription)

	if strings.TrimSpace(customRules) != "" && customRules != "no additional rules" {
		prompt += fmt.Sprintf(`

# Additional Rules and Constraints

%s`, customRules)
	}

	prompt += `

# Instructions

Please analyze the codebase context and complete the requested task. If you're generating code changes, provide them in git diff format so they can be applied directly to the codebase.`

	return prompt
}

// EstimateTokens estimates the number of tokens in a text string
//
// This uses a simple approximation: ~4 characters per token (average for English text).
// This is a rough estimate and may vary by model and language.
//
// Parameters:
//   - text: The text to estimate tokens for
//
// Returns:
//   - int: Estimated number of tokens (always >= 0)
func (a *App) EstimateTokens(text string) int {
	// Validate input
	if text == "" {
		return 0
	}

	// Simple approximation: ~4 characters per token
	// This is a rough estimate used by many LLM providers
	tokens := len(text) / 4

	// Ensure non-negative result
	if tokens < 0 {
		return 0
	}

	return tokens
}

// EstimateCost estimates the cost of an LLM API call
//
// This calculates the estimated cost based on the provider, model, and token count.
// Uses the latest pricing as of October 2025.
//
// Parameters:
//   - provider: LLM provider (google, openai, anthropic, custom)
//   - model: Model name
//   - inputTokens: Number of input tokens
//   - outputTokens: Estimated number of output tokens
//
// Returns:
//   - float64: Estimated cost in USD (always >= 0.0)
func (a *App) EstimateCost(provider, model string, inputTokens, outputTokens int) float64 {
	// Validate inputs
	if inputTokens < 0 {
		runtime.LogWarningf(a.ctx, "EstimateCost called with negative inputTokens: %d", inputTokens)
		inputTokens = 0
	}

	if outputTokens < 0 {
		runtime.LogWarningf(a.ctx, "EstimateCost called with negative outputTokens: %d", outputTokens)
		outputTokens = 0
	}

	if strings.TrimSpace(provider) == "" {
		runtime.LogWarning(a.ctx, "EstimateCost called with empty provider")
		return 0.0
	}

	if strings.TrimSpace(model) == "" {
		runtime.LogWarningf(a.ctx, "EstimateCost called with empty model for provider: %s", provider)
		model = "unknown"
	}

	var inputCostPer1M, outputCostPer1M float64

	switch provider {
	case "google":
		if strings.Contains(model, "flash") {
			inputCostPer1M = 0.075
			outputCostPer1M = 0.30
		} else {
			// Pro model
			inputCostPer1M = 1.25
			outputCostPer1M = 10.0
		}

	case "openai":
		if strings.Contains(model, "nano") {
			inputCostPer1M = 0.05
			outputCostPer1M = 0.40
		} else if strings.Contains(model, "mini") {
			inputCostPer1M = 0.25
			outputCostPer1M = 2.00
		} else {
			// GPT-5 full
			inputCostPer1M = 1.25
			outputCostPer1M = 10.00
		}

	case "anthropic":
		inputCostPer1M = 3.0
		outputCostPer1M = 15.0

	case "custom":
		// Unknown pricing for custom providers
		return 0.0

	default:
		runtime.LogWarningf(a.ctx, "Unknown provider '%s' for cost estimation", provider)
		return 0.0
	}

	inputCost := float64(inputTokens) / 1_000_000.0 * inputCostPer1M
	outputCost := float64(outputTokens) / 1_000_000.0 * outputCostPer1M
	totalCost := inputCost + outputCost

	// Ensure non-negative result
	if totalCost < 0 {
		return 0.0
	}

	return totalCost
}

// countProcessableItems estimates the total number of operations for progress tracking.
// Operations: 1 for root dir line, 1 for each dir/file entry in tree, 1 for each file content read.
func (a *App) countProcessableItems(jobCtx context.Context, rootDir string, excludedMap map[string]bool) (int, error) {
	count := 1 // For the root directory line itself

	var counterHelper func(currentPath string) error
	counterHelper = func(currentPath string) error {
		select {
		case <-jobCtx.Done():
			return jobCtx.Err()
		default:
		}

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			runtime.LogWarningf(a.ctx, "countProcessableItems: error reading dir %s: %v", currentPath, err)
			return nil // Continue counting other parts if a subdir is inaccessible
		}

		for _, entry := range entries {
			path := filepath.Join(currentPath, entry.Name())
			relPath, _ := filepath.Rel(rootDir, path)

			if excludedMap[relPath] {
				continue
			}

			count++ // For the tree entry (dir or file)

			if entry.IsDir() {
				err := counterHelper(path)
				if err != nil { // Propagate cancellation or critical errors
					return err
				}
			} else {
				count++ // For reading the file content
			}
		}
		return nil
	}

	err := counterHelper(rootDir)
	if err != nil {
		return 0, err // Return error if counting was interrupted (e.g. context cancelled)
	}
	return count, nil
}

type generationProgressState struct {
	processedItems int
	totalItems     int
}

func (a *App) emitProgress(state *generationProgressState) {
	runtime.EventsEmit(a.ctx, "shotgunContextGenerationProgress", map[string]int{
		"current": state.processedItems,
		"total":   state.totalItems,
	})
}

// generateShotgunOutputWithProgress generates the TXT output with progress reporting and size limits
func (a *App) generateShotgunOutputWithProgress(jobCtx context.Context, rootDir string, excludedPaths []string) (string, error) {
	if err := jobCtx.Err(); err != nil { // Check for cancellation at the beginning
		return "", err
	}

	excludedMap := make(map[string]bool)
	for _, p := range excludedPaths {
		excludedMap[p] = true
	}

	totalItems, err := a.countProcessableItems(jobCtx, rootDir, excludedMap)
	if err != nil {
		return "", fmt.Errorf("failed to count processable items: %w", err)
	}
	progressState := &generationProgressState{processedItems: 0, totalItems: totalItems}
	a.emitProgress(progressState) // Initial progress (0 / total)

	var output strings.Builder
	var fileContents strings.Builder

	// Root directory line - no size limit enforced
	output.WriteString(filepath.Base(rootDir) + string(os.PathSeparator) + "\n")
	progressState.processedItems++
	a.emitProgress(progressState)

	// buildShotgunTreeRecursive is a recursive helper for generating the tree string and file contents
	var buildShotgunTreeRecursive func(pCtx context.Context, currentPath, prefix string) error
	buildShotgunTreeRecursive = func(pCtx context.Context, currentPath, prefix string) error {
		select {
		case <-pCtx.Done():
			return pCtx.Err()
		default:
		}

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			runtime.LogWarningf(a.ctx, "buildShotgunTreeRecursive: error reading dir %s: %v", currentPath, err)
			// Decide if this error should halt the entire process or just skip this directory
			// For now, returning nil to skip, but log it. Could also return the error.
			return nil // Or return err if this should stop everything
		}

		// Sort entries like in ListFiles for consistent tree
		sort.SliceStable(entries, func(i, j int) bool {
			entryI := entries[i]
			entryJ := entries[j]
			isDirI := entryI.IsDir()
			isDirJ := entryJ.IsDir()
			if isDirI && !isDirJ {
				return true
			}
			if !isDirI && isDirJ {
				return false
			}
			return strings.ToLower(entryI.Name()) < strings.ToLower(entryJ.Name())
		})

		// Create a temporary slice to hold non-excluded entries for correct prefixing
		var visibleEntries []fs.DirEntry
		for _, entry := range entries {
			path := filepath.Join(currentPath, entry.Name())
			relPath, _ := filepath.Rel(rootDir, path)
			if !excludedMap[relPath] {
				visibleEntries = append(visibleEntries, entry)
			}
		}

		for i, entry := range visibleEntries {
			select {
			case <-pCtx.Done():
				return pCtx.Err()
			default:
			}

			path := filepath.Join(currentPath, entry.Name())
			relPath, _ := filepath.Rel(rootDir, path)

			isLast := i == len(visibleEntries)-1

			branch := "|-- "
			nextPrefix := prefix + "|   "
			if isLast {
				branch = "`-- "
				nextPrefix = prefix + "    "
			}
			output.WriteString(prefix + branch + entry.Name() + "\n")

			progressState.processedItems++ // For tree entry
			a.emitProgress(progressState)

			// No size limit check - allow unlimited context generation

			if entry.IsDir() {
				err := buildShotgunTreeRecursive(pCtx, path, nextPrefix)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return err
					}
					fmt.Printf("Error processing subdirectory %s: %v\n", path, err)
				}
			} else {
				select { // Check before heavy I/O
				case <-pCtx.Done():
					return pCtx.Err()
				default:
				}

				// Detect if file is binary before reading
				isBinary, err := isBinaryFile(path)
				if err != nil {
					runtime.LogWarningf(a.ctx, "Error detecting binary for %s: %v (skipping)", path, err)
					progressState.processedItems++ // Count as processed
					a.emitProgress(progressState)
					continue // Skip this file
				}

				// Skip binary files in context generation
				if isBinary {
					runtime.LogDebugf(a.ctx, "Skipping binary file in context: %s", relPath)
					// Add a placeholder comment in the file contents section
					relPathForwardSlash := filepath.ToSlash(relPath)
					fileContents.WriteString(fmt.Sprintf("<!-- Binary file skipped: %s -->\n", relPathForwardSlash))
					progressState.processedItems++ // Count as processed
					a.emitProgress(progressState)
					continue // Skip to next file
				}

				// Read file content
				content, err := os.ReadFile(path)
				if err != nil {
					runtime.LogWarningf(a.ctx, "Error reading file %s: %v", path, err)
					// Include error message in output for debugging
					relPathForwardSlash := filepath.ToSlash(relPath)
					fileContents.WriteString(fmt.Sprintf("<file path=\"%s\">\n", relPathForwardSlash))
					fileContents.WriteString(fmt.Sprintf("Error reading file: %v", err))
					fileContents.WriteString("\n</file>\n")
					progressState.processedItems++
					a.emitProgress(progressState)
					continue
				}

				// Validate UTF-8 encoding
				if !utf8.Valid(content) {
					runtime.LogWarningf(a.ctx, "File contains invalid UTF-8 (skipping): %s", relPath)
					relPathForwardSlash := filepath.ToSlash(relPath)
					fileContents.WriteString(fmt.Sprintf("<!-- File skipped (invalid UTF-8): %s -->\n", relPathForwardSlash))
					progressState.processedItems++
					a.emitProgress(progressState)
					continue
				}

				// Ensure forward slashes for the name attribute, consistent with documentation.
				relPathForwardSlash := filepath.ToSlash(relPath)

				fileContents.WriteString(fmt.Sprintf("<file path=\"%s\">\n", relPathForwardSlash))
				fileContents.WriteString(string(content))
				fileContents.WriteString("\n</file>\n") // Each file block ends with a newline

				progressState.processedItems++ // For file content
				a.emitProgress(progressState)

				// No size limit check - allow unlimited context generation
			}
		}
		return nil
	}

	err = buildShotgunTreeRecursive(jobCtx, rootDir, "")
	if err != nil {
		return "", fmt.Errorf("failed to build tree for shotgun: %w", err)
	}

	if err := jobCtx.Err(); err != nil { // Check for cancellation before final string operations
		return "", err
	}

	// The final output is the tree, a newline, then all concatenated file contents.
	// If fileContents is empty, we still want the newline after the tree.
	// If fileContents is not empty, it already ends with a newline, so an extra one might not be desired
	// depending on how it's structured. Given each <file> block ends with \n, this should be fine.
	return output.String() + "\n" + strings.TrimRight(fileContents.String(), "\n"), nil
}

// ============================================================================
// Watchman - File System Watcher
// ============================================================================

// Watchman monitors file system changes and emits events to the frontend
// It uses fsnotify to watch for file/directory changes in real-time
//
// Key Features:
// - Real-time file system monitoring using fsnotify
// - Recursive directory watching
// - Gitignore and custom ignore pattern support
// - Debounced event emission to prevent UI flooding
// - Thread-safe start/stop operations
//
// Events Emitted:
// - "file-tree-changed": When files/directories are added, modified, or deleted
type Watchman struct {
	app         *App               // Reference to main app for emitting events
	rootDir     string             // Root directory being watched
	fsWatcher   *fsnotify.Watcher  // fsnotify watcher instance
	watchedDirs map[string]bool    // Tracks directories explicitly added to fsnotify
	mu          sync.Mutex         // Protects concurrent access to watcher state
	cancelFunc  context.CancelFunc // Function to cancel the watcher goroutine

	// Ignore patterns used during file scanning
	currentProjectGitignore *gitignore.GitIgnore // Compiled .gitignore patterns for the project
	currentCustomPatterns   *gitignore.GitIgnore // Compiled custom ignore patterns
}

// NewWatchman creates a new Watchman instance
//
// Parameters:
//   - app: Reference to the main App for emitting events
//
// Returns:
//   - *Watchman: New watchman instance (not yet started)
func NewWatchman(app *App) *Watchman {
	return &Watchman{
		app:         app,
		watchedDirs: make(map[string]bool),
	}
}

// StartFileWatcher is called by JavaScript to start watching a directory.
func (a *App) StartFileWatcher(rootDirPath string) error {
	runtime.LogInfof(a.ctx, "StartFileWatcher called for: %s", rootDirPath)
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}
	return a.fileWatcher.Start(rootDirPath)
}

// StopFileWatcher is called by JavaScript to stop the current watcher.
func (a *App) StopFileWatcher() error {
	runtime.LogInfo(a.ctx, "StopFileWatcher called")
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}
	a.fileWatcher.Stop()
	return nil
}

func (w *Watchman) Start(newRootDir string) error {
	w.Stop() // Stop any existing watcher

	w.mu.Lock()
	w.rootDir = newRootDir
	if w.rootDir == "" {
		w.mu.Unlock()
		runtime.LogInfo(w.app.ctx, "Watchman: Root directory is empty, not starting.")
		return nil
	}
	w.mu.Unlock()

	// Initialize patterns based on App's current state
	if w.app.useGitignore {
		w.currentProjectGitignore = w.app.projectGitignore
	} else {
		w.currentProjectGitignore = nil
	}
	if w.app.useCustomIgnore {
		w.currentCustomPatterns = w.app.currentCustomIgnorePatterns
	} else {
		w.currentCustomPatterns = nil
	}

	w.mu.Lock()
	// Ensure settings are loaded if they haven't been (e.g. if called before startup completes, though unlikely)
	// However, loadSettings is called in startup, so this should generally be populated.
	ctx, cancel := context.WithCancel(w.app.ctx) // Use app's context as parent
	w.cancelFunc = cancel
	w.mu.Unlock()

	var err error
	w.fsWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		runtime.LogErrorf(w.app.ctx, "Watchman: Error creating fsnotify watcher: %v", err)
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}
	w.watchedDirs = make(map[string]bool) // Initialize/clear

	runtime.LogInfof(w.app.ctx, "Watchman: Starting for directory %s", newRootDir)
	w.addPathsToWatcherRecursive(newRootDir) // Add initial paths

	go w.run(ctx)
	return nil
}

func (w *Watchman) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancelFunc != nil {
		runtime.LogInfo(w.app.ctx, "Watchman: Stopping...")
		w.cancelFunc()
		w.cancelFunc = nil // Allow GC and prevent double-cancel
	}
	if w.fsWatcher != nil {
		err := w.fsWatcher.Close()
		if err != nil {
			runtime.LogWarningf(w.app.ctx, "Watchman: Error closing fsnotify watcher: %v", err)
		}
		w.fsWatcher = nil
	}
	w.rootDir = ""
	w.watchedDirs = make(map[string]bool) // Clear watched directories
}

func (w *Watchman) run(ctx context.Context) {
	defer func() {
		if w.fsWatcher != nil {
			// This close is a safeguard; Stop() should ideally be called.
			w.fsWatcher.Close()
		}
		runtime.LogInfo(w.app.ctx, "Watchman: Goroutine stopped.")
	}()

	w.mu.Lock()
	currentRootDir := w.rootDir
	w.mu.Unlock()
	runtime.LogInfof(w.app.ctx, "Watchman: Monitoring goroutine started for %s", currentRootDir)

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			shutdownRootDir := w.rootDir // Re-fetch rootDir under lock as it might have changed
			w.mu.Unlock()
			runtime.LogInfof(w.app.ctx, "Watchman: Context cancelled, shutting down watcher for %s.", shutdownRootDir)
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				runtime.LogInfo(w.app.ctx, "Watchman: fsnotify events channel closed.")
				return
			}
			runtime.LogDebugf(w.app.ctx, "Watchman: fsnotify event: %s", event)

			w.mu.Lock()
			currentRootDir = w.rootDir // Update currentRootDir under lock
			// Safely copy ignore patterns
			projIgn := w.currentProjectGitignore
			custIgn := w.currentCustomPatterns
			w.mu.Unlock()

			if currentRootDir == "" { // Watcher might have been stopped
				continue
			}

			relEventPath, err := filepath.Rel(currentRootDir, event.Name)
			if err != nil {
				runtime.LogWarningf(w.app.ctx, "Watchman: Could not get relative path for event %s (root: %s): %v", event.Name, currentRootDir, err)
				continue
			}

			// Check if the event path is ignored
			isIgnoredByGit := projIgn != nil && projIgn.MatchesPath(relEventPath)
			isIgnoredByCustom := custIgn != nil && custIgn.MatchesPath(relEventPath)

			if isIgnoredByGit || isIgnoredByCustom {
				runtime.LogDebugf(w.app.ctx, "Watchman: Ignoring event for %s as it's an ignored path.", event.Name)
				continue
			}

			// Handle relevant events (excluding Chmod)
			if event.Op&fsnotify.Chmod == 0 {
				runtime.LogInfof(w.app.ctx, "Watchman: Relevant change detected for %s in %s", event.Name, currentRootDir)
				w.app.notifyFileChange(currentRootDir)
			}

			// Dynamic directory watching
			if event.Op&fsnotify.Create != 0 {
				info, statErr := os.Stat(event.Name)
				if statErr == nil && info.IsDir() {
					// Check if this new directory itself is ignored before adding
					isNewDirIgnoredByGit := projIgn != nil && projIgn.MatchesPath(relEventPath)
					isNewDirIgnoredByCustom := custIgn != nil && custIgn.MatchesPath(relEventPath)
					if !isNewDirIgnoredByGit && !isNewDirIgnoredByCustom {
						runtime.LogDebugf(w.app.ctx, "Watchman: New directory created %s, adding to watcher.", event.Name)
						w.addPathsToWatcherRecursive(event.Name) // This will add event.Name and its children
					} else {
						runtime.LogDebugf(w.app.ctx, "Watchman: New directory %s is ignored, not adding to watcher.", event.Name)
					}
				}
			}

			if event.Op&fsnotify.Remove != 0 || event.Op&fsnotify.Rename != 0 {
				w.mu.Lock()
				if w.watchedDirs[event.Name] {
					runtime.LogDebugf(w.app.ctx, "Watchman: Watched directory %s removed/renamed, removing from watcher.", event.Name)
					// fsnotify might remove it automatically, but explicit removal is safer for our tracking
					if w.fsWatcher != nil { // Check fsWatcher as it might be closed by Stop()
						err := w.fsWatcher.Remove(event.Name)
						if err != nil {
							runtime.LogWarningf(w.app.ctx, "Watchman: Error removing path %s from fsnotify: %v", event.Name, err)
						}
					}
					delete(w.watchedDirs, event.Name)
				}
				w.mu.Unlock()
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				runtime.LogInfo(w.app.ctx, "Watchman: fsnotify errors channel closed.")
				return
			}
			runtime.LogErrorf(w.app.ctx, "Watchman: fsnotify error: %v", err)
		}
	}
}

func (w *Watchman) addPathsToWatcherRecursive(baseDirToAdd string) {
	w.mu.Lock() // Lock to access watcher and ignore patterns
	fsW := w.fsWatcher
	projIgn := w.currentProjectGitignore
	custIgn := w.currentCustomPatterns
	overallRoot := w.rootDir
	w.mu.Unlock()

	if fsW == nil || overallRoot == "" {
		runtime.LogWarningf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: fsWatcher is nil or rootDir is empty. Skipping add for %s.", baseDirToAdd)
		return
	}

	filepath.WalkDir(baseDirToAdd, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			runtime.LogWarningf(w.app.ctx, "Watchman scan error accessing %s: %v", path, walkErr)
			if d != nil && d.IsDir() && path != overallRoot { // Changed scanRootDir to overallRoot for clarity
				return filepath.SkipDir
			}
			return nil // Try to continue
		}

		if !d.IsDir() {
			return nil
		}

		relPath, errRel := filepath.Rel(overallRoot, path)
		if errRel != nil {
			runtime.LogWarningf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: Could not get relative path for %s (root: %s): %v", path, overallRoot, errRel)
			return nil // Continue with other paths
		}

		// Skip .git directory at the top level of overallRoot
		if d.IsDir() && d.Name() == ".git" {
			parentDir := filepath.Dir(path)
			if parentDir == overallRoot {
				runtime.LogDebugf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: Skipping .git directory: %s", path)
				return filepath.SkipDir
			}
		}

		isIgnoredByGit := projIgn != nil && projIgn.MatchesPath(relPath)
		isIgnoredByCustom := custIgn != nil && custIgn.MatchesPath(relPath)

		if isIgnoredByGit || isIgnoredByCustom {
			runtime.LogDebugf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: Skipping ignored directory: %s", path)
			return filepath.SkipDir
		}

		errAdd := fsW.Add(path)
		if errAdd != nil {
			runtime.LogWarningf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: Error adding path %s to fsnotify: %v", path, errAdd)
		} else {
			runtime.LogDebugf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: Added to watcher: %s", path)
			w.mu.Lock()
			w.watchedDirs[path] = true
			w.mu.Unlock()
		}
		return nil
	})
}

// notifyFileChange is an internal method for the App to emit a Wails event.
func (a *App) notifyFileChange(rootDir string) {
	runtime.EventsEmit(a.ctx, "projectFilesChanged", rootDir)
}

// RefreshIgnoresAndRescan is called when ignore settings change in the App.
func (w *Watchman) RefreshIgnoresAndRescan() error {
	w.mu.Lock()
	if w.rootDir == "" {
		w.mu.Unlock()
		runtime.LogInfo(w.app.ctx, "Watchman.RefreshIgnoresAndRescan: No rootDir, skipping.")
		return nil
	}
	runtime.LogInfo(w.app.ctx, "Watchman.RefreshIgnoresAndRescan: Refreshing ignore patterns and re-scanning.")

	// Update patterns based on App's current state
	if w.app.useGitignore {
		w.currentProjectGitignore = w.app.projectGitignore
	} else {
		w.currentProjectGitignore = nil
	}
	if w.app.useCustomIgnore {
		w.currentCustomPatterns = w.app.currentCustomIgnorePatterns
	} else {
		w.currentCustomPatterns = nil
	}
	currentRootDir := w.rootDir
	defer w.mu.Unlock()

	// Stop existing watcher (closes, clears watchedDirs)
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}
	w.watchedDirs = make(map[string]bool)

	// Create new watcher
	var err error
	w.fsWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		runtime.LogErrorf(w.app.ctx, "Watchman.RefreshIgnoresAndRescan: Error creating new fsnotify watcher: %v", err)
		return fmt.Errorf("failed to create new fsnotify watcher: %w", err)
	}

	w.addPathsToWatcherRecursive(currentRootDir) // Add paths with new rules
	w.app.notifyFileChange(currentRootDir)       // Notify frontend to refresh its view

	return nil
}

// --- Configuration Management ---

func (a *App) compileCustomIgnorePatterns() error {
	if strings.TrimSpace(a.settings.CustomIgnoreRules) == "" {
		a.currentCustomIgnorePatterns = nil
		runtime.LogDebug(a.ctx, "Custom ignore rules are empty, no patterns compiled.")
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(a.settings.CustomIgnoreRules, "\r\n", "\n"), "\n")
	var validLines []string
	for _, line := range lines {
		// CompileIgnoreLines should handle empty/comment lines appropriately based on .gitignore syntax
		validLines = append(validLines, line)
	}

	ign := gitignore.CompileIgnoreLines(validLines...)
	// Поскольку CompileIgnoreLines в этой версии не возвращает ошибку,
	// проверка на err удалена.
	// Если ign будет nil (например, если все строки были пустыми или комментариями,
	// и библиотека так обрабатывает), то это будет корректно обработано ниже.
	a.currentCustomIgnorePatterns = ign
	runtime.LogInfo(a.ctx, "Successfully compiled custom ignore patterns.")
	return nil
}

// loadSettings loads user settings from the config file
// If the file doesn't exist or can't be read, it uses default embedded settings
//
// This function is called during app startup to restore user preferences
// It handles various error cases gracefully and always ensures valid settings
func (a *App) loadSettings() {
	// Start with default embedded rules as fallback
	a.settings.CustomIgnoreRules = defaultCustomIgnoreRulesContent

	// If config path is not set, we can't load from disk
	if a.configPath == "" {
		runtime.LogWarningf(a.ctx, "Config path is empty, using default custom ignore rules (embedded).")
		if err := a.compileCustomIgnorePatterns(); err != nil {
			// Error already logged in compileCustomIgnorePatterns
		}
		return
	}

	// Try to read the settings file
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - this is normal for first run
			runtime.LogInfo(a.ctx, "Settings file not found. Using default custom ignore rules (embedded) and attempting to save them.")
			// Create the settings file with defaults
			if errSave := a.saveSettings(); errSave != nil {
				runtime.LogErrorf(a.ctx, "Failed to save default settings: %v", errSave)
			}
		} else {
			// Some other error reading the file
			runtime.LogErrorf(a.ctx, "Error reading settings file %s: %v. Using default custom ignore rules (embedded).", a.configPath, err)
		}
	} else {
		// Successfully read the file, try to parse it
		err = json.Unmarshal(data, &a.settings)
		if err != nil {
			// JSON parsing failed - use defaults
			runtime.LogErrorf(a.ctx, "Error unmarshalling settings from %s: %v. Using default custom ignore rules (embedded).", a.configPath, err)
			a.settings.CustomIgnoreRules = defaultCustomIgnoreRulesContent
		} else {
			// Successfully loaded settings
			runtime.LogInfo(a.ctx, "Successfully loaded custom ignore rules from config.")

			// If loaded rules are empty, fall back to defaults
			if strings.TrimSpace(a.settings.CustomIgnoreRules) == "" && strings.TrimSpace(defaultCustomIgnoreRulesContent) != "" {
				runtime.LogInfo(a.ctx, "Loaded custom ignore rules are empty, falling back to default embedded rules.")
				a.settings.CustomIgnoreRules = defaultCustomIgnoreRulesContent
			}

			// Ensure custom prompt rules have a default value
			if strings.TrimSpace(a.settings.CustomPromptRules) == "" {
				runtime.LogInfo(a.ctx, "Custom prompt rules are empty or missing, using default.")
				a.settings.CustomPromptRules = defaultCustomPromptRulesContent
			}
		}
	}

	// Compile the ignore patterns (whether from file or defaults)
	if errCompile := a.compileCustomIgnorePatterns(); errCompile != nil {
		// Error already logged in compileCustomIgnorePatterns
	}
}

// saveSettings saves the current settings to the config file
// Creates the config directory if it doesn't exist
//
// Returns:
//   - error: Error if saving fails, nil on success
func (a *App) saveSettings() error {
	// Can't save if we don't have a config path
	if a.configPath == "" {
		err := errors.New("config path is not set, cannot save settings")
		runtime.LogError(a.ctx, err.Error())
		return err
	}

	// Convert settings to JSON with pretty formatting
	data, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		runtime.LogErrorf(a.ctx, "Error marshalling settings: %v", err)
		return err
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(a.configPath)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		runtime.LogErrorf(a.ctx, "Error creating config directory %s: %v", configDir, err)
		return err
	}

	err = os.WriteFile(a.configPath, data, 0644)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Error writing settings to %s: %v", a.configPath, err)
		return err
	}
	runtime.LogInfo(a.ctx, "Settings saved successfully.")
	return nil
}

// GetCustomIgnoreRules returns the current custom ignore rules as a string.
func (a *App) GetCustomIgnoreRules() string {
	// Ensure settings are loaded if they haven't been (e.g. if called before startup completes, though unlikely)
	// However, loadSettings is called in startup, so this should generally be populated.
	return a.settings.CustomIgnoreRules
}

// SetCustomIgnoreRules updates the custom ignore rules, saves them, and recompiles.
func (a *App) SetCustomIgnoreRules(rules string) error {
	a.settings.CustomIgnoreRules = rules
	// Attempt to compile first. If compilation fails, we might not want to save invalid rules,
	// or save them and let the user know they are not effective.
	// For now, compile then save. If compile fails, the old patterns (or nil) remain active.
	compileErr := a.compileCustomIgnorePatterns()

	saveErr := a.saveSettings()
	if saveErr != nil {
		return fmt.Errorf("failed to save settings: %w (compile error: %v)", saveErr, compileErr)
	}
	if compileErr != nil {
		return fmt.Errorf("rules saved, but failed to compile custom ignore patterns: %w", compileErr)
	}

	if a.fileWatcher != nil && a.fileWatcher.rootDir != "" {
		return a.fileWatcher.RefreshIgnoresAndRescan()
	}
	return nil
}

// GetCustomPromptRules returns the current custom prompt rules as a string.
func (a *App) GetCustomPromptRules() string {
	if strings.TrimSpace(a.settings.CustomPromptRules) == "" {
		return defaultCustomPromptRulesContent
	}
	return a.settings.CustomPromptRules
}

// SetCustomPromptRules updates the custom prompt rules and saves them.
func (a *App) SetCustomPromptRules(rules string) error {
	a.settings.CustomPromptRules = rules
	err := a.saveSettings()
	if err != nil {
		return fmt.Errorf("failed to save custom prompt rules: %w", err)
	}
	runtime.LogInfo(a.ctx, "Custom prompt rules saved successfully.")
	return nil
}

// SetUseGitignore updates the app's setting for using .gitignore and informs the watcher.
func (a *App) SetUseGitignore(enabled bool) error {
	a.useGitignore = enabled
	runtime.LogInfof(a.ctx, "App setting useGitignore changed to: %v", enabled)
	if a.fileWatcher != nil && a.fileWatcher.rootDir != "" {
		// Assuming watcher is for the current project if active.
		return a.fileWatcher.RefreshIgnoresAndRescan()
	}
	return nil
}

// SetUseCustomIgnore updates the app's setting for using custom ignore rules and informs the watcher.
func (a *App) SetUseCustomIgnore(enabled bool) error {
	a.useCustomIgnore = enabled
	runtime.LogInfof(a.ctx, "App setting useCustomIgnore changed to: %v", enabled)
	if a.fileWatcher != nil && a.fileWatcher.rootDir != "" {
		// Assuming watcher is for the current project if active.
		return a.fileWatcher.RefreshIgnoresAndRescan()
	}
	return nil
}

// ============================================================================
// Clipboard Management - WSL Support
// ============================================================================

// WSLClipboardSetText copies text to clipboard using PowerShell Set-Clipboard for WSL compatibility
//
// This method solves the clipboard problem in WSL (Windows Subsystem for Linux) environments
// where the standard X11/WSLg clipboard integration may not work reliably.
//
// Strategy:
// - For small text (<10KB): Use direct PowerShell command with escaped text
// - For large text (>=10KB): Write to temp file and read via PowerShell to avoid command line limits
//
// This is part of a 3-tier clipboard fallback system:
// 1. WSL → Windows clipboard (this function)
// 2. Wails clipboard API (cross-platform)
// 3. Browser clipboard API (last resort)
//
// Parameters:
//   - text: Text to copy to clipboard
//
// Returns:
//   - error: Error if not in WSL or if clipboard operation fails
func (a *App) WSLClipboardSetText(text string) error {
	// Check if we're in WSL by looking for WSL environment variables
	wslDistro := os.Getenv("WSL_DISTRO_NAME")
	if wslDistro == "" {
		// Not in WSL, fall back to regular Wails clipboard
		return fmt.Errorf("not running in WSL environment, use regular clipboard methods")
	}

	runtime.LogInfof(a.ctx, "Using WSL clipboard via PowerShell Set-Clipboard for %d characters", len(text))

	// For small text (<10KB), use direct command approach, otherwise use temp file
	const maxDirectArgLength = 10000

	if len(text) <= maxDirectArgLength {
		// For smaller text, try direct command approach first
		// Escape single quotes by doubling them (PowerShell escaping)
		escapedText := strings.ReplaceAll(text, "'", "''")
		cmd := exec.Command("powershell.exe", "-Command", "Set-Clipboard -Value '"+escapedText+"'")

		err := cmd.Run()
		if err != nil {
			runtime.LogErrorf(a.ctx, "Failed to copy to clipboard via PowerShell Set-Clipboard (direct): %v", err)
			// Fallback to temp file even for small data if direct method fails
			return a.wslClipboardViaTempFile(text)
		}

		runtime.LogInfo(a.ctx, "Successfully copied to Windows clipboard via PowerShell Set-Clipboard (direct)")
		return nil
	}

	// For any text larger than 10KB, always use temporary file approach
	// This avoids command line argument length limits
	runtime.LogInfof(a.ctx, "Text size %d > %d, using temporary file method", len(text), maxDirectArgLength)
	return a.wslClipboardViaTempFile(text)
}

// wslClipboardViaTempFile handles large clipboard data by writing to a temporary file
// This avoids PowerShell command line argument length limits
//
// Process:
// 1. Write text to a temp file in WSL /tmp directory
// 2. Convert WSL path to Windows path (e.g., /tmp/file.txt → \\wsl$\Ubuntu\tmp\file.txt)
// 3. Use PowerShell Get-Content to read the file and pipe to Set-Clipboard
// 4. Clean up the temp file
//
// Parameters:
//   - text: Text to copy to clipboard
//
// Returns:
//   - error: Error if file operations or clipboard operation fails
func (a *App) wslClipboardViaTempFile(text string) error {
	// Create a temporary file in WSL /tmp directory (Linux path)
	// Use timestamp for uniqueness
	timestamp := time.Now().UnixNano()
	tempFileName := fmt.Sprintf("shotgun_clip_%d.txt", timestamp)

	// Write to WSL /tmp directory (accessible from Go/Linux)
	wslTempFilePath := filepath.Join("/tmp", tempFileName)

	runtime.LogInfof(a.ctx, "Using temporary file for large clipboard data: %s", wslTempFilePath)

	// Write text to temporary file with UTF-8 encoding
	err := os.WriteFile(wslTempFilePath, []byte(text), 0644)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to write temporary clipboard file: %v", err)
		return fmt.Errorf("failed to write temporary clipboard file: %w", err)
	}

	// Ensure cleanup of temporary file (using WSL path)
	defer func() {
		if removeErr := os.Remove(wslTempFilePath); removeErr != nil {
			runtime.LogWarningf(a.ctx, "Failed to clean up temporary clipboard file %s: %v", wslTempFilePath, removeErr)
		}
	}()

	// Get WSL distro name for Windows path conversion
	wslDistro := os.Getenv("WSL_DISTRO_NAME")
	if wslDistro == "" {
		// Fallback to common distro name or try generic approach
		wslDistro = "Ubuntu"
		runtime.LogWarningf(a.ctx, "WSL_DISTRO_NAME not found, using fallback: %s", wslDistro)
	}

	// Convert WSL path to Windows-accessible path: \\wsl$\distro\tmp\file.txt
	winAccessiblePath := fmt.Sprintf("\\\\wsl$\\%s\\tmp\\%s", wslDistro, tempFileName)
	runtime.LogInfof(a.ctx, "PowerShell will access file via: %s", winAccessiblePath)

	// Use PowerShell to read file from WSL filesystem and set clipboard
	psCommand := fmt.Sprintf("Get-Content -Path '%s' -Encoding UTF8 -Raw | Set-Clipboard", winAccessiblePath)
	cmd := exec.Command("powershell.exe", "-Command", psCommand)

	err = cmd.Run()
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to copy to clipboard via PowerShell Set-Clipboard (temp file): %v", err)
		// Try alternative WSL localhost path if \\wsl$ failed
		winAccessiblePathAlt := fmt.Sprintf("\\\\wsl.localhost\\%s\\tmp\\%s", wslDistro, tempFileName)
		runtime.LogInfof(a.ctx, "Retrying with alternative path: %s", winAccessiblePathAlt)
		psCommandAlt := fmt.Sprintf("Get-Content -Path '%s' -Encoding UTF8 -Raw | Set-Clipboard", winAccessiblePathAlt)
		cmdAlt := exec.Command("powershell.exe", "-Command", psCommandAlt)

		err = cmdAlt.Run()
		if err != nil {
			runtime.LogErrorf(a.ctx, "Both WSL path methods failed: %v", err)
			return fmt.Errorf("failed to copy to Windows clipboard via temp file: %w", err)
		}
	}

	runtime.LogInfo(a.ctx, "Successfully copied to Windows clipboard via PowerShell Set-Clipboard (temp file)")
	return nil
}
