<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <!-- Screen title with file count -->
    <div class="mb-6">
      <h1 class="text-3xl font-bold">Choose Files</h1>
      <p class="text-gray-600 mt-2">
        {{ selectedFileCount }} of {{ totalFileCount }} files selected
      </p>
    </div>

    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-4">
        Select which files to include in your context. Use search and quick actions for faster selection.
      </p>

      <!--
        Search bar with ref for keyboard shortcut focus
        Ctrl+F will focus this input
      -->
      <div class="mb-4">
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          placeholder="Search files... (Ctrl+F)"
          class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          @input="handleSearchInput"
        />
        <!-- Clear search button (shown when search is active) -->
        <button
          v-if="searchQuery"
          @click="clearSearch"
          class="absolute right-8 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
        >
          ✕
        </button>
      </div>

      <!--
        Quick actions toolbar
        Provides one-click operations for common file selection tasks
      -->
      <div class="flex gap-2 mb-4 flex-wrap">
        <button
          @click="selectAll"
          class="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 transition-colors"
          title="Select all files (Ctrl+A)"
        >
          ✓ Select All
        </button>
        <button
          @click="deselectAll"
          class="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 transition-colors"
          title="Deselect all files (Ctrl+Shift+A)"
        >
          ✗ Deselect All
        </button>
        <button
          @click="invertSelection"
          class="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 transition-colors"
          title="Invert selection (Ctrl+I)"
        >
          ⇄ Invert
        </button>
        <button
          @click="collapseAll"
          class="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 transition-colors"
        >
          ▶ Collapse All
        </button>
        <button
          @click="expandAll"
          class="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 transition-colors"
        >
          ▼ Expand All
        </button>
        <button
          @click="copyFileList"
          class="px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition-colors ml-auto"
          title="Copy file tree and contents to clipboard"
        >
          📋 Copy Files & Contents
        </button>
      </div>

      <!--
        Enhanced file tree component
        Shows hierarchical file structure with checkboxes, icons, and search highlighting
      -->
      <div class="border rounded p-4 bg-gray-50 min-h-[300px] max-h-[500px] overflow-auto">
        <EnhancedFileTree
          v-if="fileTreeNodes && fileTreeNodes.length > 0"
          :nodes="fileTreeNodes"
          :search-query="searchQuery"
          @toggle-exclude="handleToggleExclude"
          @toggle-expand="handleToggleExpand"
        />
        <div v-else-if="loading" class="text-center py-8">
          <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p class="text-gray-500 mt-2">Loading file tree...</p>
        </div>
        <div v-else-if="error" class="text-center py-8">
          <p class="text-red-500">{{ error }}</p>
          <button
            @click="loadFileTree"
            class="mt-2 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Retry
          </button>
        </div>
        <div v-else class="text-center py-12">
          <div class="text-6xl mb-4">📂</div>
          <h3 class="text-lg font-semibold text-gray-700 mb-2">Empty Folder</h3>
          <p class="text-gray-500 mb-4">
            This folder appears to be empty or contains no processable files.
          </p>
          <p class="text-sm text-gray-400 mb-4">
            All files may be excluded by .gitignore or custom ignore patterns.
          </p>
          <button
            @click="handleBack"
            class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            ← Select Different Folder
          </button>
        </div>
      </div>

      <!--
        Statistics and warnings panel
        Shows file counts, sizes, and warnings for binary files
      -->
      <div class="mt-4 space-y-2">
        <!-- Size indicator -->
        <div class="p-3 rounded border border-blue-200 bg-blue-50">
          <div class="flex justify-between text-sm mb-1">
            <span>Context size:</span>
            <span class="font-semibold">{{ formatSize(currentSize) }}</span>
          </div>
          <div class="flex justify-between text-xs text-gray-600">
            <span>{{ selectedFileCount }} files selected</span>
            <span v-if="binaryFileCount > 0" class="text-purple-600">
              {{ binaryFileCount }} binary files (will be skipped)
            </span>
          </div>
        </div>

        <!-- Warning for large context -->
        <div
          v-if="currentSize > 50 * 1024 * 1024"
          class="p-3 rounded border border-yellow-300 bg-yellow-50"
        >
          <p class="text-sm text-yellow-800">
            ⚠️ Large context detected ({{ formatSize(currentSize) }}).
            This may take longer to process and could exceed LLM token limits.
          </p>
        </div>

        <!-- Warning for binary files -->
        <div
          v-if="binaryFileCount > 0 && selectedFileCount > 0"
          class="p-3 rounded border border-purple-300 bg-purple-50"
        >
          <p class="text-sm text-purple-800">
            🔒 {{ binaryFileCount }} binary file{{ binaryFileCount > 1 ? 's' : '' }} selected.
            Binary files will be automatically skipped during context generation.
          </p>
        </div>
      </div>
    </div>

    <!-- Navigation buttons -->
    <div class="flex justify-between mt-8">
      <button
        @click="handleBack"
        class="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
      >
        ← Back
      </button>
      <button
        @click="handleNext"
        class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        :disabled="selectedFileCount === 0"
        :class="{ 'opacity-50 cursor-not-allowed': selectedFileCount === 0 }"
      >
        Next →
      </button>
    </div>
  </div>
</template>

<script setup>
/**
 * FileSelectionScreen Component
 *
 * Allows users to select which files to include in the context generation.
 * Features:
 * - Enhanced file tree with tri-state checkboxes
 * - Real-time search filtering
 * - Quick actions (select all, deselect, invert, collapse/expand)
 * - Size indicator with warnings (50%, 80%, 90%)
 * - Keyboard shortcuts (Ctrl+F, Ctrl+A, Ctrl+I, etc.)
 * - Integration with Wails backend for file listing
 */

import { ref, computed, onMounted, nextTick } from 'vue';
import { navigateTo, navigateBack } from '../router';
import EnhancedFileTree from '../components/EnhancedFileTree.vue';
import { useFileSelectionShortcuts, useKeyboardShortcuts } from '../composables/useKeyboardShortcuts';
import { useAppStore } from '../stores/appStore';
import { useToast } from '../composables/useToast';

// Import Wails runtime for backend calls
const ListFiles = window.go?.main?.App?.ListFiles;
const ReadFileContents = window.go?.main?.App?.ReadFileContents;

// Get store and toast
const store = useAppStore();
const { showSuccess, showError, showWarning } = useToast();

// Get project root from store
const projectRoot = computed(() => store.projectFolder);

// ============================================================================
// Component State
// ============================================================================

/** File tree data from backend */
const fileTreeNodes = ref([]);

/** Search query for filtering files */
const searchQuery = ref('');

/** Reference to search input for keyboard shortcut focus */
const searchInput = ref(null);

/** Loading state */
const loading = ref(false);

/** Error message */
const error = ref('');

/** Current context size in bytes */
const currentSize = ref(0);

/** Map of manually toggled nodes (path -> excluded state) */
const manuallyToggledNodes = ref(new Map());

// ============================================================================
// Computed Properties
// ============================================================================

// No size limit - removed maxSize and sizePercentage calculations

/**
 * Count total number of files (not directories)
 */
const totalFileCount = computed(() => {
  return countFiles(fileTreeNodes.value);
});

/**
 * Count selected (non-excluded) files
 */
const selectedFileCount = computed(() => {
  return countSelectedFiles(fileTreeNodes.value);
});

/**
 * Count binary files in selected files
 */
const binaryFileCount = computed(() => {
  return countBinaryFiles(fileTreeNodes.value);
});

// ============================================================================
// Lifecycle Hooks
// ============================================================================

onMounted(async () => {
  // Load file tree from backend
  await loadFileTree();

  // Register keyboard shortcuts
  useKeyboardShortcuts(useFileSelectionShortcuts({
    focusSearch: () => searchInput.value?.focus(),
    selectAll,
    deselectAll,
    invertSelection,
    nextScreen: handleNext,
    clearSearch
  }));
});

// ============================================================================
// File Tree Operations
// ============================================================================

/**
 * Load file tree from backend
 * Calls Wails ListFiles method and processes the result
 *
 * This function requires a valid Wails backend connection.
 * It will display an error if the backend is not available.
 */
async function loadFileTree() {
  // Check if Wails backend is available
  if (!ListFiles || typeof ListFiles !== 'function') {
    error.value = 'Backend not available. Please ensure the application is running properly.';
    console.error('Wails backend not available - ListFiles method not found:', ListFiles);
    showError('Backend not available. Please restart the application.');
    return;
  }

  // Validate project root
  if (!projectRoot.value || typeof projectRoot.value !== 'string' || projectRoot.value.trim() === '') {
    showError('No project folder selected. Please go back and select a folder.');
    error.value = 'No project folder selected.';
    return;
  }

  loading.value = true;
  error.value = '';

  try {
    // Call backend to list files
    const nodes = await ListFiles(projectRoot.value);

    // Validate response
    if (!nodes) {
      throw new Error('No data received from backend');
    }

    if (!Array.isArray(nodes)) {
      throw new Error('Invalid response format (expected array)');
    }

    // Process nodes: add expanded property, merge with manual toggles
    const processedNodes = processFileNodes(nodes);

    // Validate processed nodes
    if (!Array.isArray(processedNodes)) {
      throw new Error('Failed to process file nodes');
    }

    fileTreeNodes.value = processedNodes;

    // Save to store (store will validate)
    store.setFileTree(nodes);

    // Calculate initial size
    calculateContextSize();

    // Check if folder is empty or all files are excluded
    if (processedNodes.length === 0) {
      showInfo('Folder loaded, but no processable files found. All files may be excluded by ignore patterns.');
    } else if (processedNodes.length === 1 && processedNodes[0].isDir && (!processedNodes[0].children || processedNodes[0].children.length === 0)) {
      showInfo('Folder loaded, but appears to be empty or all files are excluded.');
    } else {
      showSuccess('File tree loaded successfully!');
    }

  } catch (err) {
    console.error('Error loading file tree:', err);
    const errorMsg = err?.message || 'Unknown error';
    error.value = `Failed to load file tree: ${errorMsg}`;
    showError(`Failed to load file tree: ${errorMsg}`);
    fileTreeNodes.value = [];
  } finally {
    loading.value = false;
  }
}

/**
 * Process file nodes from backend
 * Adds UI-specific properties and merges with manual toggles
 *
 * @param {Array} nodes - Raw nodes from backend
 * @returns {Array} Processed nodes with UI properties
 */
function processFileNodes(nodes) {
  // Validate input
  if (!nodes || !Array.isArray(nodes)) {
    console.error('Invalid nodes for processing:', nodes);
    return [];
  }

  try {
    return nodes.map(node => {
      // Validate node
      if (!node || typeof node !== 'object') {
        console.error('Invalid node:', node);
        return null;
      }

      // Ensure manuallyToggledNodes is a Map
      if (!(manuallyToggledNodes.value instanceof Map)) {
        manuallyToggledNodes.value = new Map();
      }

      // Check if this node was manually toggled
      const manualToggle = manuallyToggledNodes.value.get(node.path);

      // Determine excluded state
      // Priority: manual toggle > gitignore > custom ignore
      const excluded = manualToggle !== undefined
        ? manualToggle
        : (node.isGitignored || node.isCustomIgnored);

      // Create processed node
      const processedNode = {
        ...node,
        excluded,
        expanded: node.isDir ? false : undefined, // Directories start collapsed
      manuallyExcluded: manualToggle === true
    };

    // Recursively process children
    if (node.children && Array.isArray(node.children) && node.children.length > 0) {
      processedNode.children = processFileNodes(node.children);
    }

    return processedNode;
  }).filter(node => node !== null); // Remove any null nodes from validation failures
  } catch (error) {
    console.error('Error processing file nodes:', error);
    return [];
  }
}

/**
 * Handle checkbox toggle for a node
 * Updates exclusion state and propagates to children
 *
 * @param {Object} node - The node that was toggled
 */
function handleToggleExclude(node) {
  // Toggle the excluded state
  const newExcludedState = !node.excluded;

  // Store in manual toggles map
  manuallyToggledNodes.value.set(node.path, newExcludedState);

  // Update the node
  node.excluded = newExcludedState;
  node.manuallyExcluded = newExcludedState;

  // If it's a directory, propagate to all children
  if (node.isDir && node.children) {
    propagateExclusionToChildren(node, newExcludedState);
  }

  // Recalculate context size
  calculateContextSize();
}

/**
 * Propagate exclusion state to all children recursively
 *
 * @param {Object} node - Parent node
 * @param {boolean} excluded - Exclusion state to propagate
 */
function propagateExclusionToChildren(node, excluded) {
  if (!node.children) return;

  for (const child of node.children) {
    child.excluded = excluded;
    manuallyToggledNodes.value.set(child.path, excluded);

    if (child.isDir && child.children) {
      propagateExclusionToChildren(child, excluded);
    }
  }
}

/**
 * Handle expand/collapse toggle for a directory
 *
 * @param {Object} node - The directory node
 */
function handleToggleExpand(node) {
  if (node.isDir) {
    node.expanded = !node.expanded;
  }
}

/**
 * Calculate total context size based on selected files
 * Updates currentSize ref
 */
function calculateContextSize() {
  currentSize.value = calculateNodeSize(fileTreeNodes.value);
}

/**
 * Recursively calculate size of selected files in a node tree
 *
 * @param {Array} nodes - Array of nodes
 * @returns {number} Total size in bytes
 */
function calculateNodeSize(nodes) {
  if (!nodes) return 0;

  let total = 0;

  for (const node of nodes) {
    if (node.excluded) continue; // Skip excluded nodes

    if (node.isDir) {
      // For directories, sum children
      total += calculateNodeSize(node.children);
    } else {
      // For files, add file size
      total += node.size || 0;
    }
  }

  return total;
}

/**
 * Count total number of files (not directories) in tree
 *
 * @param {Array} nodes - Array of nodes
 * @returns {number} Total file count
 */
function countFiles(nodes) {
  if (!nodes) return 0;

  let count = 0;

  for (const node of nodes) {
    if (node.isDir) {
      count += countFiles(node.children);
    } else {
      count++;
    }
  }

  return count;
}

/**
 * Count selected (non-excluded) files in tree
 *
 * @param {Array} nodes - Array of nodes
 * @returns {number} Selected file count
 */
function countSelectedFiles(nodes) {
  if (!nodes) return 0;

  let count = 0;

  for (const node of nodes) {
    if (node.excluded) continue;

    if (node.isDir) {
      count += countSelectedFiles(node.children);
    } else {
      count++;
    }
  }

  return count;
}


// ============================================================================
// Quick Actions
// ============================================================================

/**
 * Select all files in the tree
 * Sets all nodes to non-excluded state
 */
function selectAll() {
  setAllNodesExcluded(fileTreeNodes.value, false);
  calculateContextSize();
}

/**
 * Deselect all files in the tree
 * Sets all nodes to excluded state
 */
function deselectAll() {
  setAllNodesExcluded(fileTreeNodes.value, true);
  calculateContextSize();
}

/**
 * Invert current selection
 * Excluded nodes become included, included nodes become excluded
 */
function invertSelection() {
  invertNodesExcluded(fileTreeNodes.value);
  calculateContextSize();
}

/**
 * Collapse all directories in the tree
 */
function collapseAll() {
  setAllNodesExpanded(fileTreeNodes.value, false);
}

/**
 * Expand all directories in the tree
 */
function expandAll() {
  setAllNodesExpanded(fileTreeNodes.value, true);
}

/**
 * Copy file tree and contents to clipboard
 * This creates a formatted output with file tree structure and file contents
 * ready for pasting into LLM conversations
 */
async function copyFileList() {
  try {
    // Validate that we have files selected
    const selectedFiles = getSelectedFilePaths(fileTreeNodes.value);

    if (!selectedFiles || selectedFiles.length === 0) {
      showWarning('No files selected. Please select at least one file.');
      return;
    }

    // Validate backend method is available
    if (!ReadFileContents) {
      showError('Backend method not available. Please restart the application.');
      console.error('ReadFileContents method is not available');
      return;
    }

    // Validate project root
    if (!projectRoot.value) {
      showError('Project root is not set.');
      return;
    }

    // Show loading state
    loading.value = true;

    // Call backend to read file contents
    const results = await ReadFileContents(projectRoot.value, selectedFiles);

    // Validate results
    if (!results || !Array.isArray(results)) {
      throw new Error('Invalid response from backend');
    }

    // Build output with file tree and contents
    let output = '';

    // Add header
    output += '# File Tree and Contents\n\n';
    output += `Project: ${projectRoot.value}\n`;
    output += `Files: ${selectedFiles.length}\n`;
    output += `Generated: ${new Date().toISOString()}\n\n`;

    // Add file tree
    output += '## File Tree\n\n';
    output += '```\n';
    output += buildFileTreeString(fileTreeNodes.value);
    output += '```\n\n';

    // Add file contents
    output += '## File Contents\n\n';

    let successCount = 0;
    let binaryCount = 0;
    let errorCount = 0;

    for (const result of results) {
      // Validate result object
      if (!result || typeof result !== 'object') {
        console.warn('Invalid result object:', result);
        continue;
      }

      const { path, content, size, isBinary, error } = result;

      if (error) {
        // File had an error
        output += `### ${path}\n\n`;
        output += `**Error:** ${error}\n\n`;
        errorCount++;
      } else if (isBinary) {
        // File is binary - skip content
        output += `### ${path}\n\n`;
        output += `**Binary file** (${formatBytes(size || 0)})\n\n`;
        binaryCount++;
      } else if (content !== undefined && content !== null) {
        // File has content
        output += `### ${path}\n\n`;
        output += '```\n';
        output += content;
        output += '\n```\n\n';
        successCount++;
      } else {
        // Unexpected case
        output += `### ${path}\n\n`;
        output += '**No content available**\n\n';
        errorCount++;
      }
    }

    // Add summary footer
    output += '---\n\n';
    output += `**Summary:** ${successCount} text files, ${binaryCount} binary files, ${errorCount} errors\n`;

    // Copy to clipboard
    await navigator.clipboard.writeText(output);

    // Show success message with details
    showSuccess(`Copied ${successCount} files to clipboard! (${binaryCount} binary files skipped, ${errorCount} errors)`);

  } catch (err) {
    console.error('Failed to copy file list and contents:', err);
    showError(`Failed to copy: ${err.message || 'Unknown error'}`);
  } finally {
    loading.value = false;
  }
}

/**
 * Build a text representation of the file tree
 * Shows only non-excluded files in a tree structure
 *
 * @param {Array} nodes - Array of file tree nodes
 * @param {string} prefix - Prefix for indentation
 * @returns {string} Tree structure as text
 */
function buildFileTreeString(nodes, prefix = '') {
  if (!nodes || !Array.isArray(nodes)) {
    return '';
  }

  let output = '';

  // Filter out excluded nodes
  const visibleNodes = nodes.filter(node => !node.excluded);

  for (let i = 0; i < visibleNodes.length; i++) {
    const node = visibleNodes[i];
    const isLast = i === visibleNodes.length - 1;

    const branch = isLast ? '└── ' : '├── ';
    const nextPrefix = prefix + (isLast ? '    ' : '│   ');

    // Add node name with size for files
    if (node.isDir) {
      output += prefix + branch + node.name + '/\n';
      // Recursively add children
      if (node.children && node.children.length > 0) {
        output += buildFileTreeString(node.children, nextPrefix);
      }
    } else {
      const sizeStr = node.size ? ` (${formatBytes(node.size)})` : '';
      const binaryStr = node.isBinary ? ' [BINARY]' : '';
      output += prefix + branch + node.name + sizeStr + binaryStr + '\n';
    }
  }

  return output;
}

/**
 * Format bytes to human-readable string
 *
 * @param {number} bytes - Number of bytes
 * @returns {string} Formatted string (e.g., "1.5 KB")
 */
function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';

  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/**
 * Recursively set excluded state for all nodes
 *
 * @param {Array} nodes - Array of nodes
 * @param {boolean} excluded - Exclusion state to set
 */
function setAllNodesExcluded(nodes, excluded) {
  if (!nodes) return;

  for (const node of nodes) {
    node.excluded = excluded;
    manuallyToggledNodes.value.set(node.path, excluded);

    if (node.isDir && node.children) {
      setAllNodesExcluded(node.children, excluded);
    }
  }
}

/**
 * Recursively invert excluded state for all nodes
 *
 * @param {Array} nodes - Array of nodes
 */
function invertNodesExcluded(nodes) {
  if (!nodes) return;

  for (const node of nodes) {
    node.excluded = !node.excluded;
    manuallyToggledNodes.value.set(node.path, node.excluded);

    if (node.isDir && node.children) {
      invertNodesExcluded(node.children);
    }
  }
}

/**
 * Recursively set expanded state for all directory nodes
 *
 * @param {Array} nodes - Array of nodes
 * @param {boolean} expanded - Expansion state to set
 */
function setAllNodesExpanded(nodes, expanded) {
  if (!nodes) return;

  for (const node of nodes) {
    if (node.isDir) {
      node.expanded = expanded;

      if (node.children) {
        setAllNodesExpanded(node.children, expanded);
      }
    }
  }
}

/**
 * Count binary files in the tree (only non-excluded files)
 *
 * @param {Array} nodes - Array of nodes
 * @returns {number} Count of binary files
 */
function countBinaryFiles(nodes) {
  if (!nodes || !Array.isArray(nodes)) {
    return 0;
  }

  let count = 0;

  for (const node of nodes) {
    // Skip excluded nodes
    if (node.excluded) continue;

    if (node.isDir) {
      // Recursively count in children
      if (node.children && Array.isArray(node.children)) {
        count += countBinaryFiles(node.children);
      }
    } else {
      // Count binary files
      if (node.isBinary) {
        count++;
      }
    }
  }

  return count;
}

/**
 * Get list of selected file paths
 *
 * @param {Array} nodes - Array of nodes
 * @returns {Array<string>} Array of file paths
 */
function getSelectedFilePaths(nodes) {
  if (!nodes || !Array.isArray(nodes)) {
    console.error('Invalid nodes for getSelectedFilePaths:', nodes);
    return [];
  }

  const paths = [];

  try {
    for (const node of nodes) {
      if (!node || typeof node !== 'object') continue;
      if (node.excluded) continue;

      if (node.isDir) {
        if (node.children && Array.isArray(node.children)) {
          paths.push(...getSelectedFilePaths(node.children));
        }
      } else {
        const path = node.relPath || node.path;
        if (path && typeof path === 'string') {
          paths.push(path);
        }
      }
    }
  } catch (error) {
    console.error('Error getting selected file paths:', error);
  }

  return paths;
}

/**
 * Get all excluded file paths from the tree
 * Used for saving exclusions to global state
 *
 * @param {Array} nodes - Array of nodes
 * @returns {Array<string>} Array of excluded file paths
 */
function getExcludedFilePaths(nodes) {
  if (!nodes || !Array.isArray(nodes)) {
    console.error('Invalid nodes for getExcludedFilePaths:', nodes);
    return [];
  }

  const paths = [];

  try {
    for (const node of nodes) {
      if (!node || typeof node !== 'object') continue;

      if (node.excluded) {
        // Node is excluded - add it
        if (node.isDir) {
          // For directories, recursively get all file paths within
          const getAllPaths = (n) => {
            const p = [];
            if (!n || typeof n !== 'object') return p;

            if (!n.isDir) {
              const path = n.relPath || n.path;
              if (path && typeof path === 'string') {
                p.push(path);
              }
            } else if (n.children && Array.isArray(n.children)) {
              for (const child of n.children) {
                p.push(...getAllPaths(child));
              }
            }
            return p;
          };
          paths.push(...getAllPaths(node));
        } else {
          const path = node.relPath || node.path;
          if (path && typeof path === 'string') {
            paths.push(path);
          }
        }
      } else if (node.isDir && node.children && Array.isArray(node.children)) {
        // Not excluded, but check children
        paths.push(...getExcludedFilePaths(node.children));
      }
    }
  } catch (error) {
    console.error('Error getting excluded file paths:', error);
  }

  return paths;
}

// ============================================================================
// Search Operations
// ============================================================================

/**
 * Handle search input with debouncing
 * Debouncing prevents excessive re-renders while typing
 */
let searchDebounceTimer = null;
function handleSearchInput() {
  // Clear existing timer
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer);
  }

  // Set new timer (300ms debounce)
  searchDebounceTimer = setTimeout(() => {
    // Search is handled by computed property in EnhancedFileTree
    // This is just for potential future analytics or logging
    console.log('Search query:', searchQuery.value);
  }, 300);
}

/**
 * Clear search query and refocus search input
 */
function clearSearch() {
  searchQuery.value = '';
  searchInput.value?.focus();
}

// ============================================================================
// UI Helper Functions
// ============================================================================

/**
 * Format size in bytes to human-readable format
 * No size limit enforced - this is for display purposes only
 *
 * @param {number} bytes - Size in bytes
 * @returns {string} Formatted size string
 */
function formatSize(bytes) {
  if (!bytes || bytes === 0) return '0 MB';

  const mb = bytes / (1024 * 1024);
  return `${mb.toFixed(2)} MB`;
}

// ============================================================================
// Navigation
// ============================================================================

/**
 * Navigate back to previous screen
 */
function handleBack() {
  navigateBack();
}

/**
 * Navigate to next screen (Mode Selection)
 * Only allowed if at least one file is selected
 */
function handleNext() {
  // Validate file count
  if (!selectedFileCount.value || selectedFileCount.value === 0) {
    showWarning('Please select at least one file before proceeding.');
    return;
  }

  // Validate file tree exists
  if (!Array.isArray(fileTreeNodes.value) || fileTreeNodes.value.length === 0) {
    showError('File tree not loaded. Please go back and select a folder again.');
    return;
  }

  // Check if only binary files are selected
  const nonBinaryCount = selectedFileCount.value - binaryFileCount.value;
  if (nonBinaryCount === 0 && binaryFileCount.value > 0) {
    showWarning('Only binary files are selected. Binary files will be skipped in context generation. Please select at least one text file.');
    return;
  }

  // Warn about large context size
  if (currentSize.value > 100 * 1024 * 1024) { // 100MB
    const sizeStr = formatSize(currentSize.value);
    console.warn(`Large context size: ${sizeStr}`);
    // Don't block, just log - user can proceed if they want
  }

  try {
    // Save selected/excluded files to global state
    // The file tree is already saved, but we need to save the selection state
    const selectedPaths = getSelectedFilePaths(fileTreeNodes.value);
    const excludedPaths = getExcludedFilePaths(fileTreeNodes.value);

    // Validate paths are arrays
    if (!Array.isArray(selectedPaths) || !Array.isArray(excludedPaths)) {
      showError('Failed to process file selections.');
      return;
    }

    // Validate we have valid paths
    if (selectedPaths.length === 0) {
      showError('No valid file paths found. Please try selecting files again.');
      return;
    }

    // Update store with selections (store will validate each path)
    selectedPaths.forEach(path => {
      if (path && typeof path === 'string') {
        store.toggleFileSelection(path, true);
      }
    });

    excludedPaths.forEach(path => {
      if (path && typeof path === 'string') {
        store.toggleFileExclusion(path, true);
      }
    });

    // Log summary for debugging
    console.log(`Proceeding with ${selectedPaths.length} files (${binaryFileCount.value} binary, ${nonBinaryCount} text)`);

    navigateTo('mode');
  } catch (error) {
    console.error('Error saving file selections:', error);
    showError('Failed to save file selections. Please try again.');
  }
}

// ============================================================================
// Component Lifecycle
// ============================================================================

// Load file tree when component is mounted
// This ensures the file tree is populated when the user navigates to this screen
onMounted(() => {
  loadFileTree();
});
</script>

<style scoped>
/**
 * FileSelectionScreen Styles
 * Additional component-specific styles
 */

/* Smooth transitions for all interactive elements */
button {
  @apply transition-all duration-200;
}

/* Loading spinner animation */
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.animate-spin {
  animation: spin 1s linear infinite;
}

/* Search input clear button positioning */
.relative {
  position: relative;
}
</style>
