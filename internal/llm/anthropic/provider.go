// Package anthropic provides a client implementation for the Anthropic API.
//
// API Reference: https://docs.anthropic.com/en/api/messages
// Authentication: providers.anthropic.api_key or ANTHROPIC_API_KEY environment variable
//
// Example usage:
//   client := anthropic.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, anthropic.WithTemperature(0.7))
//
// Anthropic models include: claude-3-5-haiku-latest, claude-sonnet-4-0, claude-opus-4-0

package anthropic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// Provider implements the unified registry
type Provider struct{}

// ensure Provider implements the common provider interface
var _ common.Provider = (*Provider)(nil)

// New creates a new Anthropic provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Providers.Anthropic.APIKey == "" {
		return nil, fmt.Errorf(`Anthropic API key is required.

You can set the API key using the environment variable ANTHROPIC_API_KEY or via slop config set anthropic-key=<your_api_key>
Get an API key from https://console.anthropic.com/settings/keys`)
	}

	// create client options
	var opts []common.ClientOption
	if cfg.Providers.Anthropic.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.Anthropic.BaseUrl))
	}
	if logger != nil {
		opts = append(opts, common.WithLogger(logger))
	}
	maxRetries := cfg.Parameters.MaxRetries
	if maxRetries > 5 {
		maxRetries = 5 // enforce maximum limit
	}
	if maxRetries > 0 {
		opts = append(opts, common.WithMaxRetries(maxRetries))
	}

	adapterClient := common.NewAdapterClient(p, cfg.Providers.Anthropic.APIKey, "https://api.anthropic.com/v1", opts...)
	return adapterClient, nil
}

// BuildOptions creates Anthropic-specific generation options from configuration
func (p *Provider) BuildOptions(cfg *config.Config) []interface{} {
	var functionalOpts []GenerateOption

	// handle system prompt from config
	if cfg.Parameters.SystemPrompt != "" {
		functionalOpts = append(functionalOpts, WithSystem(cfg.Parameters.SystemPrompt))
	}

	if cfg.Parameters.Temperature > 0 {
		functionalOpts = append(functionalOpts, WithTemperature(cfg.Parameters.Temperature))
	}
	if cfg.Parameters.MaxTokens > 0 {
		functionalOpts = append(functionalOpts, WithMaxTokens(cfg.Parameters.MaxTokens))
	}
	if cfg.Parameters.TopP > 0 {
		functionalOpts = append(functionalOpts, WithTopP(cfg.Parameters.TopP))
	}
	if len(cfg.Parameters.StopSequences) > 0 {
		functionalOpts = append(functionalOpts, WithStopSequences(cfg.Parameters.StopSequences))
	}
	if cfg.Format.JSON {
		functionalOpts = append(functionalOpts, WithJSONFormat())
	}

	return []interface{}{NewGenerateOptions(functionalOpts...)}
}

// RequiresAPIKey returns true; Anthropic requires an API key
func (p *Provider) RequiresAPIKey() bool {
	return true
}

// returns the name of this provider
func (p *Provider) ProviderName() string {
	return "anthropic"
}

// BuildRequest creates an Anthropic-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// convert options to Anthropic-specific options
	var config *GenerateOptions
	if options != nil {
		if anthropicOpts, ok := options.(*GenerateOptions); ok {
			config = anthropicOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request using common utilities
	common.LogAPIRequest(logger, "Anthropic", modelName, messages, &config.GenerateOptions)

	// separate system messages from user/assistant messages
	var systemPrompt string
	var filteredMessages []common.Message

	for _, msg := range messages {
		if msg.Role == "system" {
			if systemPrompt == "" {
				systemPrompt = msg.Content
			} else {
				systemPrompt += "\n\n" + msg.Content
			}
		} else {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	// use system prompt from config if no system messages found
	if systemPrompt == "" && config.System != "" {
		systemPrompt = config.System
	}

	// create Anthropic-specific request payload
	requestBody := &MessagesRequest{
		Model:     modelName,
		Messages:  filteredMessages,
		MaxTokens: 1024,                  // Anthropic requires max_tokens, so we set a default
		Stream:    common.BoolPtr(false), // Disable streaming for now
	}

	// set system prompt if provided
	if systemPrompt != "" {
		requestBody.System = systemPrompt
	}

	// map common generation options to Anthropic's API format
	if config.Temperature != nil {
		requestBody.Temperature = config.Temperature
	}
	if config.MaxTokens != nil {
		requestBody.MaxTokens = *config.MaxTokens
	}
	if config.TopP != nil {
		requestBody.TopP = config.TopP
	}

	// map Anthropic-specific options
	if config.TopK != nil {
		requestBody.TopK = config.TopK
	}
	if len(config.StopSequences) > 0 {
		requestBody.StopSequences = config.StopSequences
	}

	return requestBody, nil
}

// ParseResponse parses an Anthropic API response and extracts content and usage
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// parse the response using Anthropic's Messages API format
	var anthropicResp MessagesResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	// extract content from the content array
	if len(anthropicResp.Content) == 0 {
		return "", nil, fmt.Errorf("no content in Anthropic response")
	}

	// concatenate all text content items
	var contentParts []string
	for _, item := range anthropicResp.Content {
		if item.Type == "text" {
			contentParts = append(contentParts, item.Text)
		}
	}

	if len(contentParts) == 0 {
		return "", nil, fmt.Errorf("no text content in Anthropic response")
	}

	content := strings.Join(contentParts, "")

	// convert Anthropic usage to common format
	var usage *common.Usage
	if anthropicResp.Usage.InputTokens > 0 || anthropicResp.Usage.OutputTokens > 0 {
		usage = &common.Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		}
	}

	// return content and usage information
	return content, usage, nil
}

// HandleError creates Anthropic-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// without the body, we can sometimes provide specific, actionable error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(`Anthropic API authentication failed.

Check your API key and ensure it is set correctly. 
You can set the API key using the environment variable ANTHROPIC_API_KEY or via slop config set anthropic-key=<your_api_key>
Get an API key from https://console.anthropic.com/settings/keys`)

	case http.StatusTooManyRequests:
		return fmt.Errorf(`Anthropic API rate limit exceeded.

Please try again later or check your limits at https://console.anthropic.com/settings/limits`)
	}

	// attempt to parse Anthropic's error format
	var errorResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		// FALLBACK if the response was not the expected JSON format
		return fmt.Errorf("Anthropic API request failed with status %d: %s", statusCode, string(body))
	}

	// now we can return a specific error message
	if errorResp.Error.Message != "" {
		return fmt.Errorf("Anthropic API error: %s", errorResp.Error.Message)
	}

	// final catch-all if parsing succeeded but the message was empty
	return fmt.Errorf("an unknown API error occurred (status %d)", statusCode)
}

// HandleConnectionError handles connection failures - for cloud services, return original error
func (p *Provider) HandleConnectionError(err error) error {
	return err
}

// Anthropic uses /v1/messages endpoint and requires specific headers
func (p *Provider) CustomizeRequest(req *http.Request) error {
	if strings.HasSuffix(req.URL.Path, "/chat/completions") {
		// handle both "/chat/completions" and "/v1/chat/completions"
		if strings.HasSuffix(req.URL.Path, "/v1/chat/completions") {
			req.URL.Path = strings.Replace(req.URL.Path, "/v1/chat/completions", "/v1/messages", 1)
		} else {
			req.URL.Path = strings.Replace(req.URL.Path, "/chat/completions", "/v1/messages", 1)
		}
	}

	// Anthropic requires x-api-key header instead of Authorization Bearer
	// see: https://docs.anthropic.com/en/api/overview#authentication
	authHeader := req.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		req.Header.Del("Authorization")
		req.Header.Set("x-api-key", apiKey)
	}

	// Anthropic requires specific API version headers
	req.Header.Set("anthropic-version", "2023-06-01")

	return nil
}
