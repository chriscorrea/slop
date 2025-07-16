// Package together provides a client implementation for the Together.AI API.
//
// API Reference: https://docs.together.ai/reference/chat-completions
// Authentication: providers.together.api_key or TOGETHER_API_KEY environment variable
//
// Example usage:
//   client := together.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, together.WithTemperature(0.7))
//
// Together model documentation: https://api.together.ai/models and https://docs.together.ai/docs/models

package together

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// Provider implements the unified registry.Provider interface for Together.AI
type Provider struct{}

// ensure Provider implements the common.Provider interface
var _ common.Provider = (*Provider)(nil)

// New creates a new Together.AI provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Providers.Together.APIKey == "" {
		return nil, fmt.Errorf(`Together.AI API key is required.

You can set the API key using the environment variable TOGETHER_API_KEY or via slop config set together-key=<your_api_key>
Get an API key from https://api.together.ai/settings/api-keys`)
	}

	// create client options
	var opts []common.ClientOption
	if cfg.Providers.Together.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.Together.BaseUrl))
	}
	if logger != nil {
		opts = append(opts, common.WithLogger(logger))
	}
	maxRetries := cfg.Parameters.MaxRetries
	if maxRetries > 5 {
		maxRetries = 5 // enforce max limit
	}
	if maxRetries > 0 {
		opts = append(opts, common.WithMaxRetries(maxRetries))
	}

	adapterClient := common.NewAdapterClient(p, cfg.Providers.Together.APIKey, "https://api.together.xyz/v1", opts...)
	return adapterClient, nil
}

// BuildOptions creates TogetherAI-specific generation options from configuration
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
	if cfg.Format.JSON {
		functionalOpts = append(functionalOpts, WithJSONFormat())
	}

	return []interface{}{NewGenerateOptions(functionalOpts...)}
}

// RequiresAPIKey returns true since TogetherAI requires an API key
func (p *Provider) RequiresAPIKey() bool {
	return true
}

// ProviderName returns the name of this provider
func (p *Provider) ProviderName() string {
	return "together"
}

// BuildRequest creates a Together.AI-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// convert options to Together-specific options
	var config *GenerateOptions
	if options != nil {
		if togetherOpts, ok := options.(*GenerateOptions); ok {
			config = togetherOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request
	common.LogAPIRequest(logger, "Together.AI", modelName, messages, &config.GenerateOptions)

	// create Together-specific request payload
	requestBody := &ChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   common.BoolPtr(false), // Disable streaming for now
	}

	// map common generation options to Together's API format
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

	// map Together-specific options
	if config.FrequencyPenalty != nil {
		requestBody.FrequencyPenalty = config.FrequencyPenalty
	}
	if config.PresencePenalty != nil {
		requestBody.PresencePenalty = config.PresencePenalty
	}
	if config.RepetitionPenalty != nil {
		requestBody.RepetitionPenalty = config.RepetitionPenalty
	}
	if config.MinP != nil {
		requestBody.MinP = config.MinP
	}
	if config.LogProbs != nil {
		requestBody.LogProbs = config.LogProbs
	}
	if config.TopLogProbs != nil {
		requestBody.TopLogProbs = config.TopLogProbs
	}
	if config.Echo != nil {
		requestBody.Echo = config.Echo
	}
	if config.N != nil {
		requestBody.N = config.N
	}
	if config.SafetyModel != nil {
		requestBody.SafetyModel = config.SafetyModel
	}

	// handle structured output if requested
	if config.ResponseFormat != nil {
		requestBody.ResponseFormat = &common.ResponseFormat{
			Type: config.ResponseFormat.Type,
		}
	}

	return requestBody, nil
}

// ParseResponse parses a Together.AI API response and extracts content and usage
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// parse the response using standard OpenAI-compatible format
	var chatResp common.ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal Together.AI response: %w", err)
	}

	// extract content from the first choice
	if len(chatResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in Together.AI response")
	}

	content := chatResp.Choices[0].Message.Content

	// return content and usage information
	return content, &chatResp.Usage, nil
}

// HandleError creates Together.AI-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// without the body, we can sometimes provide specific, actionable error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(`Together.AI API authentication failed.

Check your API key and ensure it is set correctly. 
You can set the API key using the environment variable TOGETHER_API_KEY or via slop config set together-key=<your_api_key>
Get an API key from https://api.together.ai/settings/api-keys`)

	case http.StatusNotFound:
		return fmt.Errorf(`Together.AI model not found.

Please ensure you are using a valid model name.
Be sure to include the provider prefix (e.g. "deepseek-ai/DeepSeek-R1")
Available models can be found at https://api.together.ai/models`)

	case http.StatusBadRequest:
		// try to parse the error for more specific handling
		if len(body) > 0 {
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if errorResp.Error.Type == "invalid_request_error" && errorResp.Error.Param == "response_format" {
					return fmt.Errorf(`Together.AI structured output error: %s

The model you selected may not support structured output. Please check the model's capabilities.`, errorResp.Error.Message)
				}
				return fmt.Errorf("Together.AI request error: %s", errorResp.Error.Message)
			}
		}
		return fmt.Errorf("Together.AI request error: invalid request parameters")

	case http.StatusTooManyRequests:
		return fmt.Errorf(`Together.AI rate limit exceeded.

Please try again later or check your usage at https://api.together.ai/`)

	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf(`Together.AI server error (status %d).

This is likely a temporary issue, but check https://status.together.ai for updates.`, statusCode)

	case http.StatusPaymentRequired:
		return fmt.Errorf(`Your account may have insufficient credits or require payment information.
Please check your account at https://api.together.ai/`)

	default:
		// try to extract error message from response body
		if len(body) > 0 {
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
				return fmt.Errorf("Together.AI error (status %d): %s", statusCode, errorResp.Error.Message)
			}
		}
		return fmt.Errorf("Together.AI request failed with status %d", statusCode)
	}
}

// CustomizeRequest adds Together.AI-specific headers or modifies the request
func (p *Provider) CustomizeRequest(req *http.Request) error {
	// no custom headers needed
	return nil
}

// HandleConnectionError creates Together.AI-specific error messages for connection failures
func (p *Provider) HandleConnectionError(err error) error {
	return fmt.Errorf(`Failed to connect to Together.AI API.

Error: %w`, err)
}
