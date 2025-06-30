package mistral

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"slop/internal/config"
	"slop/internal/llm/common"
)

// Provider implements the unified registry.Provider interface for Mistral AI
type Provider struct{}

// ensure Provider implements the common.Provider interface
var _ common.Provider = (*Provider)(nil)

// New creates a new Mistral provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Providers.Mistral.APIKey == "" {
		return nil, fmt.Errorf(`Mistral API key is required.
		
You can set the API key using the environment variable MISTRAL_API_KEY or via slop config set mistral-key=<your_api_key>
Get an API key from https://console.mistral.ai/api-keys`)
	}

	// create client options
	var opts []common.ClientOption
	if cfg.Providers.Mistral.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.Mistral.BaseUrl))
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

	adapterClient := common.NewAdapterClient(p, cfg.Providers.Mistral.APIKey, "https://api.mistral.ai/v1", opts...)
	return adapterClient, nil
}

// BuildOptions creates Mistral-specific generation options from configuration
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
		functionalOpts = append(functionalOpts, WithRandomSeed(*cfg.Parameters.Seed))
	}
	if cfg.Format.JSON {
		functionalOpts = append(functionalOpts, WithJSONFormat())
	}

	return []interface{}{NewGenerateOptions(functionalOpts...)}
}

// RequiresAPIKey returns true since Mistral requires an API key
func (p *Provider) RequiresAPIKey() bool {
	return true
}

// ProviderName returns the name of this provider
func (p *Provider) ProviderName() string {
	return "mistral"
}

// BuildRequest creates a Mistral-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// convert options to Mistral-specific options
	var config *GenerateOptions
	if options != nil {
		if mistralOpts, ok := options.(*GenerateOptions); ok {
			config = mistralOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request using common utilities
	common.LogAPIRequest(logger, "Mistral", modelName, messages, &config.GenerateOptions)

	// create Mistral-specific request payload
	requestBody := &ChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   common.BoolPtr(false), // Disable streaming for now
	}

	// map common generation options to Mistral's API format
	if config.Temperature != nil {
		requestBody.Temperature = config.Temperature
	}
	if config.MaxTokens != nil {
		requestBody.MaxTokens = config.MaxTokens
	}
	if config.TopP != nil {
		requestBody.TopP = config.TopP
	}
	if len(config.Stop) > 0 {
		requestBody.Stop = config.Stop
	}

	// handle Mistral-specific seed field mapping
	if config.RandomSeed != nil {
		requestBody.RandomSeed = config.RandomSeed
	}

	// handle response format for structured output
	if config.ResponseFormat != nil {
		requestBody.ResponseFormat = &common.ResponseFormat{
			Type: config.ResponseFormat.Type,
		}
	}

	return requestBody, nil
}

// ParseResponse parses a Mistral API response and extracts content and usage
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// parse the response using standard OpenAI-compatible format
	var chatResp common.ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal Mistral response: %w", err)
	}

	// extract content from the first choice
	if len(chatResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in Mistral response")
	}

	content := chatResp.Choices[0].Message.Content

	// return content and usage information
	return content, &chatResp.Usage, nil
}

// HandleError creates Mistral-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// without the body, we can sometimes provide specific, actionable error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(`Mistral API authentication failed.

Check your API key and ensure it is set correctly. 
You can set the API key using the environment variable MISTRAL_API_KEY or via slop config set mistral-key=<your_api_key>
Get an API key from https://console.mistral.ai/api-keys`)

	case http.StatusTooManyRequests:
		return fmt.Errorf(`Mistral API rate limit exceeded.

Please try again later or check your usage at https://console.mistral.ai/`)
	}

	// attempt to parse the structured JSON error from the response body.
	var errorResp common.ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		// FALLBACK if the response was not the expected JSON format:
		return fmt.Errorf("Mistral API request failed with status %d: %s", statusCode, string(body))
	}

	// now we can return a much more helpful, specific error message!
	if errorResp.Error.Message != "" {
		return fmt.Errorf("Mistral API error: %s", errorResp.Error.Message)
	}

	// final catch-all if parsing succeeded but the message was empty
	return fmt.Errorf("an unknown API error occurred (status %d)", statusCode)
}

// HandleConnectionError handles connection failures - for cloud services, return original error
func (p *Provider) HandleConnectionError(err error) error {
	return err
}

// CustomizeRequest allows Mistral to customize the HTTP request if needed
func (p *Provider) CustomizeRequest(req *http.Request) error {
	// no customization needed at this time
	// this is implemented for completeness/future extensibility

	return nil
}
