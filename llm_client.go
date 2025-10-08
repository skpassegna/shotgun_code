package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

/**
 * LLM Client for Shotgun Code
 *
 * This module implements a unified client for calling various LLM APIs including
 * Google AI Studio (Gemini), OpenAI (GPT), Anthropic (Claude), and custom OpenAI-compatible APIs.
 *
 * Key Features:
 * - Unified interface for multiple LLM providers
 * - Support for custom OpenAI-compatible APIs
 * - Error handling and retries
 * - Token usage tracking
 * - Cost estimation with latest pricing (October 2025)
 * - Timeout handling
 *
 * Supported Providers (October 2025):
 * - google: Google Gemini API
 *   - gemini-2.5-flash (default): Best price/performance, 1M context, thinking capabilities
 *   - gemini-2.5-pro: Advanced reasoning, 2M context
 *   - Pricing: Flash $0.075/$0.30 per 1M tokens (input/output), Pro $1.25-$2.50/$10-$15 per 1M tokens
 *
 * - openai: OpenAI API
 *   - gpt-5-mini (default): Balanced performance, 400K context
 *   - gpt-5: Flagship model for coding/agentic tasks, 400K context
 *   - gpt-5-nano: Budget option for simple tasks, 400K context
 *   - Pricing: GPT-5 $1.25/$10, GPT-5 mini $0.25/$2.00, GPT-5 nano $0.05/$0.40 per 1M tokens
 *
 * - anthropic: Anthropic Claude API
 *   - claude-sonnet-4-5-20250929 (default): Best coding model, 200K context
 *   - Pricing: $3/$15 per 1M tokens (input/output)
 *
 * - custom: Custom OpenAI-compatible API
 *   - Configurable base URL and model name
 *   - Optional API key
 *   - Uses OpenAI chat completions format
 *
 * Security:
 * - API keys are never logged
 * - API keys are stored encrypted in local config
 * - API keys are never sent to external servers (except the LLM provider)
 */

// LLMClient handles API calls to various LLM providers
type LLMClient struct {
	app        *App         // Reference to main app for logging
	httpClient *http.Client // HTTP client with timeout
}

// LLMRequest represents a request to an LLM API
type LLMRequest struct {
	Provider    string  `json:"provider"`    // Provider: google, openai, anthropic, custom
	APIKey      string  `json:"apiKey"`      // API key for the provider (optional for custom)
	Prompt      string  `json:"prompt"`      // The prompt to send
	Model       string  `json:"model"`       // Model name (e.g., gemini-2.5-flash, gpt-5-mini, claude-sonnet-4-5-20250929)
	Temperature float64 `json:"temperature"` // Temperature (0.0-1.0)
	MaxTokens   int     `json:"maxTokens"`   // Maximum tokens to generate
	BaseURL     string  `json:"baseURL"`     // Custom base URL (for custom provider only)
}

// LLMResponse represents a response from an LLM API
type LLMResponse struct {
	Content    string  `json:"content"`    // Generated text
	TokensUsed int     `json:"tokensUsed"` // Total tokens used (prompt + completion)
	Cost       float64 `json:"cost"`       // Estimated cost in USD
	Model      string  `json:"model"`      // Model used
	Provider   string  `json:"provider"`   // Provider used
}

// NewLLMClient creates a new LLM client instance
//
// Parameters:
//   - app: Reference to the main App struct for logging
//
// Returns:
//   - *LLMClient: Initialized LLM client with 60-second timeout
func NewLLMClient(app *App) *LLMClient {
	return &LLMClient{
		app: app,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // 60-second timeout for API calls
		},
	}
}

// CallLLM calls the appropriate LLM API based on the provider
//
// This method routes the request to the appropriate provider-specific method
// and handles common error cases.
//
// Parameters:
//   - ctx: Context for cancellation
//   - req: LLM request with provider, API key, prompt, etc.
//
// Returns:
//   - *LLMResponse: Response from the LLM API
//   - error: Error if the call fails
//
// Example:
//
//	req := LLMRequest{
//	    Provider: "google",
//	    APIKey: "your-api-key",
//	    Prompt: "Write a hello world function in Go",
//	    Model: "gemini-1.5-pro",
//	    Temperature: 0.7,
//	    MaxTokens: 2048,
//	}
//	resp, err := client.CallLLM(ctx, req)
func (c *LLMClient) CallLLM(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	// Validate request
	if req.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if req.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Set default model if not specified
	if req.Model == "" {
		req.Model = c.getDefaultModel(req.Provider)
	}

	// Set default temperature if not specified
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	// Set default max tokens if not specified
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	// Route to appropriate provider
	switch req.Provider {
	case "google":
		return c.callGoogleAI(ctx, req)
	case "openai":
		return c.callOpenAI(ctx, req)
	case "anthropic":
		return c.callAnthropic(ctx, req)
	case "custom":
		return c.callCustomOpenAICompatible(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// getDefaultModel returns the default model for a provider (October 2025 latest models)
//
// Parameters:
//   - provider: Provider name (google, openai, anthropic, custom)
//
// Returns:
//   - string: Default model name
func (c *LLMClient) getDefaultModel(provider string) string {
	defaults := map[string]string{
		"google":    "gemini-2.5-flash",           // Best price/performance with thinking capabilities
		"openai":    "gpt-5-mini",                 // Balanced performance for most tasks
		"anthropic": "claude-sonnet-4-5-20250929", // Best coding model as of Oct 2025
		"custom":    "",                           // No default for custom - user must specify
	}
	return defaults[provider]
}

// callGoogleAI calls the Google AI Studio API (Gemini)
//
// API Documentation: https://ai.google.dev/api/rest
//
// Parameters:
//   - ctx: Context for cancellation
//   - req: LLM request
//
// Returns:
//   - *LLMResponse: Response from Google AI
//   - error: Error if the call fails
func (c *LLMClient) callGoogleAI(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Calling Google AI with model: %s", req.Model))

	// Build API URL
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", req.Model, req.APIKey)

	// Build request body
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": req.Prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     req.Temperature,
			"maxOutputTokens": req.MaxTokens,
		},
	}

	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract generated text
	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	generatedText := apiResp.Candidates[0].Content.Parts[0].Text

	// Calculate cost based on model (October 2025 pricing)
	// Gemini 2.5 Flash: $0.075 per 1M input tokens, $0.30 per 1M output tokens
	// Gemini 2.5 Pro: $1.25-$2.50 per 1M input tokens, $10-$15 per 1M output tokens (depends on prompt length)
	var inputCost, outputCost float64
	if strings.Contains(req.Model, "flash") {
		// Gemini 2.5 Flash pricing
		inputCost = float64(apiResp.UsageMetadata.PromptTokenCount) / 1_000_000.0 * 0.075
		outputCost = float64(apiResp.UsageMetadata.CandidatesTokenCount) / 1_000_000.0 * 0.30
	} else {
		// Gemini 2.5 Pro pricing (using lower tier for simplicity)
		inputCost = float64(apiResp.UsageMetadata.PromptTokenCount) / 1_000_000.0 * 1.25
		outputCost = float64(apiResp.UsageMetadata.CandidatesTokenCount) / 1_000_000.0 * 10.0
	}
	totalCost := inputCost + outputCost

	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Google AI response received: %d tokens, $%.6f", apiResp.UsageMetadata.TotalTokenCount, totalCost))

	return &LLMResponse{
		Content:    generatedText,
		TokensUsed: apiResp.UsageMetadata.TotalTokenCount,
		Cost:       totalCost,
		Model:      req.Model,
		Provider:   "google",
	}, nil
}

// callOpenAI calls the OpenAI API (GPT)
//
// API Documentation: https://platform.openai.com/docs/api-reference
//
// Parameters:
//   - ctx: Context for cancellation
//   - req: LLM request
//
// Returns:
//   - *LLMResponse: Response from OpenAI
//   - error: Error if the call fails
func (c *LLMClient) callOpenAI(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Calling OpenAI with model: %s", req.Model))

	// Build API URL
	url := "https://api.openai.com/v1/chat/completions"

	// Build request body
	requestBody := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}

	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract generated text
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	generatedText := apiResp.Choices[0].Message.Content

	// Calculate cost based on model (October 2025 pricing)
	// GPT-5: $1.25 per 1M input tokens, $10.00 per 1M output tokens
	// GPT-5 mini: $0.25 per 1M input tokens, $2.00 per 1M output tokens
	// GPT-5 nano: $0.05 per 1M input tokens, $0.40 per 1M output tokens
	var inputCost, outputCost float64
	if strings.Contains(req.Model, "nano") {
		// GPT-5 nano pricing
		inputCost = float64(apiResp.Usage.PromptTokens) / 1_000_000.0 * 0.05
		outputCost = float64(apiResp.Usage.CompletionTokens) / 1_000_000.0 * 0.40
	} else if strings.Contains(req.Model, "mini") {
		// GPT-5 mini pricing
		inputCost = float64(apiResp.Usage.PromptTokens) / 1_000_000.0 * 0.25
		outputCost = float64(apiResp.Usage.CompletionTokens) / 1_000_000.0 * 2.00
	} else {
		// GPT-5 (full) pricing
		inputCost = float64(apiResp.Usage.PromptTokens) / 1_000_000.0 * 1.25
		outputCost = float64(apiResp.Usage.CompletionTokens) / 1_000_000.0 * 10.00
	}
	totalCost := inputCost + outputCost

	runtime.LogInfo(c.app.ctx, fmt.Sprintf("OpenAI response received: %d tokens, $%.6f", apiResp.Usage.TotalTokens, totalCost))

	return &LLMResponse{
		Content:    generatedText,
		TokensUsed: apiResp.Usage.TotalTokens,
		Cost:       totalCost,
		Model:      req.Model,
		Provider:   "openai",
	}, nil
}

// callAnthropic calls the Anthropic API (Claude)
//
// API Documentation: https://docs.anthropic.com/claude/reference
//
// Parameters:
//   - ctx: Context for cancellation
//   - req: LLM request
//
// Returns:
//   - *LLMResponse: Response from Anthropic
//   - error: Error if the call fails
func (c *LLMClient) callAnthropic(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Calling Anthropic with model: %s", req.Model))

	// Build API URL
	url := "https://api.anthropic.com/v1/messages"

	// Build request body
	requestBody := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}

	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", req.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract generated text
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	generatedText := apiResp.Content[0].Text

	// Calculate cost (October 2025 pricing)
	// Claude Sonnet 4.5: $3 per 1M input tokens, $15 per 1M output tokens
	// This is the latest model as of September 29, 2025
	inputCost := float64(apiResp.Usage.InputTokens) / 1_000_000.0 * 3.0
	outputCost := float64(apiResp.Usage.OutputTokens) / 1_000_000.0 * 15.0
	totalCost := inputCost + outputCost
	totalTokens := apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens

	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Anthropic response received: %d tokens, $%.6f", totalTokens, totalCost))

	return &LLMResponse{
		Content:    generatedText,
		TokensUsed: totalTokens,
		Cost:       totalCost,
		Model:      req.Model,
		Provider:   "anthropic",
	}, nil
}

// callCustomOpenAICompatible calls a custom OpenAI-compatible API
//
// This function allows users to connect to any API that implements the OpenAI chat completions format.
// Examples: LocalAI, Ollama with OpenAI compatibility, LM Studio, vLLM, etc.
//
// Parameters:
//   - ctx: Context for cancellation
//   - req: LLM request with BaseURL and Model specified
//
// Returns:
//   - *LLMResponse: Response from the custom API
//   - error: Error if the call fails
func (c *LLMClient) callCustomOpenAICompatible(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Calling custom OpenAI-compatible API at %s with model: %s", req.BaseURL, req.Model))

	// Validate required fields for custom provider
	if req.BaseURL == "" {
		return nil, fmt.Errorf("baseURL is required for custom provider")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("model is required for custom provider")
	}

	// Build API URL - append /v1/chat/completions if not already present
	url := req.BaseURL
	if !strings.HasSuffix(url, "/chat/completions") && !strings.HasSuffix(url, "/v1/chat/completions") {
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		url += "v1/chat/completions"
	}

	// Build request body (OpenAI format)
	requestBody := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}

	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add API key if provided (optional for custom providers)
	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response (OpenAI format)
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract generated text
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	generatedText := apiResp.Choices[0].Message.Content

	// For custom providers, we don't calculate cost (unknown pricing)
	// Return 0 cost and let the user track it themselves
	totalTokens := apiResp.Usage.TotalTokens
	if totalTokens == 0 {
		// Some custom APIs might not return token usage
		totalTokens = apiResp.Usage.PromptTokens + apiResp.Usage.CompletionTokens
	}

	runtime.LogInfo(c.app.ctx, fmt.Sprintf("Custom API response received: %d tokens (cost not calculated for custom providers)", totalTokens))

	return &LLMResponse{
		Content:    generatedText,
		TokensUsed: totalTokens,
		Cost:       0.0, // Cost unknown for custom providers
		Model:      req.Model,
		Provider:   "custom",
	}, nil
}
