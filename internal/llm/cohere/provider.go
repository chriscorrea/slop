// Package cohere provides a client implementation for the Cohere API.
//
// API Reference: https://docs.cohere.com/v2/reference/chat
// Authentication: providers.cohere.api_key or COHERE_API_KEY environment variable
//
// Example usage:
//   client := cohere.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, cohere.WithTemperature(0.7))

package cohere

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// Provider implements the unified registry.Provider interface for Cohere AI
type Provider struct{}

// esure Provider implements the common.Provider interface
var _ common.Provider = (*Provider)(nil)

// creates a new Cohere provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	if cfg.Providers.Cohere.APIKey == "" {
		return nil, fmt.Errorf(`Cohere API key is required!
		
You can set the API key using the environment variable COHERE_API_KEY or via slop config set cohere-key=<your_api_key>
Get an API key from https://dashboard.cohere.com/api-keys`)
	}

	// create client options
	var opts []common.ClientOption
	if cfg.Providers.Cohere.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.Cohere.BaseUrl))
	}
	if logger != nil {
		opts = append(opts, common.WithLogger(logger))
	}
	maxRetries := cfg.Parameters.MaxRetries
	if maxRetries > 5 {
		maxRetries = 5 // Enforce maximum limit
	}
	if maxRetries > 0 {
		opts = append(opts, common.WithMaxRetries(maxRetries))
	}

	adapterClient := common.NewAdapterClient(p, cfg.Providers.Cohere.APIKey, "https://api.cohere.com/v2", opts...)
	return adapterClient, nil
}

// BuildOptions creates Cohere-specific generation options from configuration
func (p *Provider) BuildOptions(cfg *config.Config) []interface{} {
	var functionalOpts []GenerateOption

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
		functionalOpts = append(functionalOpts, WithStop(cfg.Parameters.StopSequences))
	}
	if cfg.Parameters.Seed != nil {
		functionalOpts = append(functionalOpts, WithSeed(*cfg.Parameters.Seed))
	}
	if cfg.Format.JSON {
		functionalOpts = append(functionalOpts, WithJSONFormat())
	}

	return []interface{}{NewGenerateOptions(functionalOpts...)}
}

// RequiresAPIKey returns true since Cohere requires an API key
func (p *Provider) RequiresAPIKey() bool {
	return true
}

// ProviderName returns the name of this provider
func (p *Provider) ProviderName() string {
	return "cohere"
}

// BuildRequest creates a Cohere-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// connvert options to Cohere-specific options
	var config *GenerateOptions
	if options != nil {
		if cohereOpts, ok := options.(*GenerateOptions); ok {
			config = cohereOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request using common utilities
	common.LogAPIRequest(logger, "Cohere", modelName, messages, &config.GenerateOptions)

	// create Cohere-specific request payload
	requestBody := &ChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   common.BoolPtr(false), // Disable streaming for now
	}

	// map common generation options to Cohere's API format
	if config.Temperature != nil {
		requestBody.Temperature = config.Temperature
	}
	if config.MaxTokens != nil {
		requestBody.MaxTokens = config.MaxTokens
	}
	if config.TopP != nil {
		requestBody.P = config.TopP // Cohere uses 'p' instead of 'top_p'
	}
	if len(config.Stop) > 0 {
		requestBody.StopSequences = config.Stop
	}

	// map Cohere-specific options
	if config.TopK != nil {
		requestBody.K = config.TopK
	}
	if config.Seed != nil {
		requestBody.Seed = config.Seed // Cohere: "determinism cannot be totally guaranteed"
	}
	if config.SafetyMode != nil {
		requestBody.SafetyMode = config.SafetyMode
	}

	// handle structured output if requested
	if config.ResponseFormat != nil {
		requestBody.ResponseFormat = &ResponseFormat{
			Type: config.ResponseFormat.Type,
		}
	}

	return requestBody, nil
}

// ParseResponse parses a Cohere API response and extracts content and usage
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// define local structs that match Cohere's actual API response format
	type cohereContentPart struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type cohereMessageArray struct {
		Role    string              `json:"role"`
		Content []cohereContentPart `json:"content"`
	}

	type cohereMessageString struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type cohereChatResponseArray struct {
		Message cohereMessageArray `json:"message"`
		Usage   Usage              `json:"usage"`
	}

	type cohereChatResponseString struct {
		Message cohereMessageString `json:"message"`
		Usage   Usage               `json:"usage"`
	}

	// try parsing as array format first (newer API format)
	var chatRespArray cohereChatResponseArray
	if err := json.Unmarshal(body, &chatRespArray); err == nil {
		// extract text content from the content array
		var content string
		if len(chatRespArray.Message.Content) == 0 {
			return "", nil, fmt.Errorf("Cohere response contained no content")
		}
		content = chatRespArray.Message.Content[0].Text

		// log token usage if available
		var usage *common.Usage
		if chatRespArray.Usage.Tokens.InputTokens > 0 {
			usage = &common.Usage{
				PromptTokens:     chatRespArray.Usage.Tokens.InputTokens,
				CompletionTokens: chatRespArray.Usage.Tokens.OutputTokens,
				TotalTokens:      chatRespArray.Usage.Tokens.InputTokens + chatRespArray.Usage.Tokens.OutputTokens,
			}
		}

		return content, usage, nil
	}

	// fall back to string format (not sure if needed, but for safety)
	var chatRespString cohereChatResponseString
	if err := json.Unmarshal(body, &chatRespString); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal Cohere response: %w", err)
	}

	// extract text content from the content string
	content := chatRespString.Message.Content

	// log token usage if available
	var usage *common.Usage
	if chatRespString.Usage.Tokens.InputTokens > 0 {
		usage = &common.Usage{
			PromptTokens:     chatRespString.Usage.Tokens.InputTokens,
			CompletionTokens: chatRespString.Usage.Tokens.OutputTokens,
			TotalTokens:      chatRespString.Usage.Tokens.InputTokens + chatRespString.Usage.Tokens.OutputTokens,
		}
	}

	// return content and usage information
	return content, usage, nil
}

// HandleError creates Cohere-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// without the body, we can provide specific, actionable error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(`Cohere API authentication failed.

Check your API key and ensure it is set correctly. 
You can set the API key using the environment variable COHERE_API_KEY or via slop config set cohere-key=<your_api_key>
Get an API key from https://dashboard.cohere.com/api-keys`)

	case http.StatusTooManyRequests:
		return fmt.Errorf(`Cohere API rate limit exceeded.

Please try again later or check your usage at https://dashboard.cohere.com/`)
	}

	// attempt to parse the structured JSON error from the response body:
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		// FALLBACK if the response was not the expected JSON format:
		return fmt.Errorf("Cohere API request failed with status %d: %s", statusCode, string(body))
	}

	// now we can return a more specific error message
	if errorResp.Message != "" {
		return fmt.Errorf("Cohere API error: %s", errorResp.Message)
	}

	// final catch-all if parsing succeeded but the message was empty
	return fmt.Errorf("an unknown API error occurred (status %d)", statusCode)
}

// HandleConnectionError handles connection failures
func (p *Provider) HandleConnectionError(err error) error {
	// connection errors are usually network issues. Return the original error as-is
	return err
}

// CustomizeRequest allows Cohere to customize the HTTP request
// Cohere uses /chat endpoint instead of /chat/completions
func (p *Provider) CustomizeRequest(req *http.Request) error {
	// update the URL if it's the standard format
	if strings.HasSuffix(req.URL.Path, "/chat/completions") {
		req.URL.Path = strings.Replace(req.URL.Path, "/chat/completions", "/chat", 1)
	}

	// Cohere's standard Bearer token authentication and JSON content
	// types are already set by CreateJSONRequest in the common package
	return nil
}
