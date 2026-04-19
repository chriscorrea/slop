// Package openai provides a client implementation for the OpenAI API.
//
// API Reference: https://platform.openai.com/docs/api-reference/chat/create
// Authentication: providers.openai.api_key or OPENAI_API_KEY environment variable
//
// Example usage:
//   client := openai.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, openai.WithTemperature(0.7))
//
// OpenAI models include: gpt-4.1-2025-04-14, o4-mini-2025-04-16, o3-2025-04-16
// OpenAI model documentation: https://platform.openai.com/docs/models

package openai

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// supportsThinking reports whether the given OpenAI model ID accepts the
// reasoning_effort parameter. Returns true for GPT-5 thinking variants and the
// o3 / o4 reasoning families; everything else (including gpt-4o) returns false
func supportsThinking(modelID string) bool {
	id := strings.ToLower(strings.TrimSpace(modelID))
	if id == "" {
		return false
	}
	switch {
	case strings.HasPrefix(id, "gpt-5.4"),
		strings.HasPrefix(id, "gpt-5"),
		strings.HasPrefix(id, "o3"),
		strings.HasPrefix(id, "o4"):
		return true
	}
	return false
}

// translateThinkingLevel maps the common ThinkingLevel to OpenAI's
// reasoning_effort string. ThinkingOff returns "" so callers can skip sending
// the field
func translateThinkingLevel(level common.ThinkingLevel) string {
	switch level {
	case common.ThinkingMedium:
		return "medium"
	case common.ThinkingHigh:
		return "high"
	default:
		return ""
	}
}

// Provider implements the unified registry.Provider interface for OpenAI
type Provider struct{}

// ensure Provider implements the common.Provider interface
var _ common.Provider = (*Provider)(nil)

// New creates a new OpenAI provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Providers.OpenAI.APIKey == "" {
		return nil, fmt.Errorf(`OpenAI API key is required.

You can set the API key using the environment variable OPENAI_API_KEY or via slop config set openai-key=<your_api_key>
Get an API key from https://platform.openai.com/api-keys`)
	}

	// create client options
	var opts []common.ClientOption
	if cfg.Providers.OpenAI.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.OpenAI.BaseUrl))
	}
	if logger != nil {
		opts = append(opts, common.WithLogger(logger))
	}
	// use provider-specific MaxRetries, fall back to global if not set
	maxRetries := cfg.Providers.OpenAI.MaxRetries
	if maxRetries == 0 {
		maxRetries = cfg.Parameters.MaxRetries
	}
	if maxRetries > 5 {
		maxRetries = 5 // enforce maximum limit
	}
	if maxRetries > 0 {
		opts = append(opts, common.WithMaxRetries(maxRetries))
	}

	adapterClient := common.NewAdapterClient(p, cfg.Providers.OpenAI.APIKey, "https://api.openai.com/v1", opts...)
	return adapterClient, nil
}

// BuildOptions creates OpenAI-specific generation options from configuration
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

	// translate thinking into reasoning_effort; BuildRequest decides whether
	// the selected model actually supports the field
	if level, err := common.ParseThinkingLevel(cfg.Parameters.Thinking); err == nil {
		if effort := translateThinkingLevel(level); effort != "" {
			functionalOpts = append(functionalOpts, WithReasoningEffort(effort))
		}
	}

	// schema-constrained structured output — ResponseSchema is pre-resolved
	// inline JSON by the config manager
	if schema := strings.TrimSpace(cfg.Parameters.ResponseSchema); schema != "" {
		functionalOpts = append(functionalOpts, withCommonSchema("response", []byte(schema)))
	}

	return []interface{}{NewGenerateOptions(functionalOpts...)}
}

// withCommonSchema adapts the common WithSchema option into the OpenAI
// GenerateOption signature so it can be applied alongside provider-specific options
func withCommonSchema(name string, schema []byte) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithSchema(name, schema)(&c.GenerateOptions)
	}
}

// RequiresAPIKey returns true since OpenAI requires an API key
func (p *Provider) RequiresAPIKey() bool {
	return true
}

// ProviderName returns the name of this provider
func (p *Provider) ProviderName() string {
	return "openai"
}

// BuildRequest creates an OpenAI-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// convert options to OpenAI-specific options
	var config *GenerateOptions
	if options != nil {
		if openaiOpts, ok := options.(*GenerateOptions); ok {
			config = openaiOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request using common utilities
	common.LogAPIRequest(logger, "OpenAI", modelName, messages, &config.GenerateOptions)

	// create OpenAI-specific request payload
	requestBody := &ChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   common.BoolPtr(false), // Disable streaming for now
	}

	// map common generation options to OpenAI's API format
	if config.Temperature != nil {
		requestBody.Temperature = config.Temperature
	}
	if config.MaxTokens != nil {
		requestBody.MaxCompletionTokens = config.MaxTokens
	}
	if config.TopP != nil {
		requestBody.TopP = config.TopP
	}
	if len(config.Stop) > 0 {
		requestBody.Stop = config.Stop
	}

	// map OpenAI-specific options
	if config.FrequencyPenalty != nil {
		requestBody.FrequencyPenalty = config.FrequencyPenalty
	}
	if config.PresencePenalty != nil {
		requestBody.PresencePenalty = config.PresencePenalty
	}
	if config.Seed != nil {
		requestBody.Seed = config.Seed
	}
	if len(config.Tools) > 0 {
		// validate every tool before sending — the API rejects requests with
		// unnamed tools or malformed parameter schemas, so fail fast with a
		// clear message here
		for i, tool := range config.Tools {
			if strings.TrimSpace(tool.Function.Name) == "" {
				return nil, fmt.Errorf("tool at index %d is missing a function name", i)
			}
			if tool.Function.Parameters != nil {
				// if caller passed raw JSON bytes, validate them directly so
				// bad JSON is surfaced by ValidateJSONSchema rather than by
				// encoding/json's stricter RawMessage check
				if raw, ok := tool.Function.Parameters.(json.RawMessage); ok {
					if err := common.ValidateJSONSchema(raw); err != nil {
						return nil, fmt.Errorf("tool %q has invalid parameters schema: %w", tool.Function.Name, err)
					}
				} else {
					paramsBytes, err := json.Marshal(tool.Function.Parameters)
					if err != nil {
						return nil, fmt.Errorf("tool %q has parameters that cannot be marshaled: %w", tool.Function.Name, err)
					}
					if err := common.ValidateJSONSchema(paramsBytes); err != nil {
						return nil, fmt.Errorf("tool %q has invalid parameters schema: %w", tool.Function.Name, err)
					}
				}
			}
		}
		requestBody.Tools = config.Tools
	}
	if config.ToolChoice != nil {
		requestBody.ToolChoice = config.ToolChoice
	}

	// reasoning_effort is only accepted by GPT-5 thinking variants and the
	// o-series. For other models silently no-op so users can keep a global
	// thinking default without errors when switching providers/models
	if config.ReasoningEffort != nil && supportsThinking(modelName) {
		requestBody.ReasoningEffort = config.ReasoningEffort
	}

	// map structured output into OpenAI's wire shape
	if config.ResponseFormat != nil {
		switch config.ResponseFormat.Type {
		case "json_schema":
			requestBody.ResponseFormat = &chatResponseFormat{
				Type: "json_schema",
				JSONSchema: &chatJSONSchemaSpec{
					Name:   config.ResponseFormat.Name,
					Schema: config.ResponseFormat.Schema,
					Strict: config.ResponseFormat.Strict,
				},
			}
		default:
			// preserve legacy json_object behavior (and any other simple type)
			requestBody.ResponseFormat = &chatResponseFormat{
				Type: config.ResponseFormat.Type,
			}
		}
	}

	return requestBody, nil
}

// ParseResponse parses an OpenAI API response and extracts content and usage
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// parse the response using standard OpenAI format
	var chatResp common.ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal OpenAI response: %w", err)
	}

	// extract content from the first choice
	if len(chatResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in OpenAI response")
	}

	content := chatResp.Choices[0].Message.Content

	// return content and usage information
	return content, &chatResp.Usage, nil
}

// HandleError creates OpenAI-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// without the body, we can sometimes provide specific, actionable error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(`OpenAI API authentication failed.

Check your API key and ensure it is set correctly. 
You can set the API key using the environment variable OPENAI_API_KEY or via slop config set openai-key=<your_api_key>
Get an API key from https://platform.openai.com/api-keys`)

	case http.StatusTooManyRequests:
		return fmt.Errorf(`OpenAI API rate limit exceeded.

Please try again later or check your usage at https://platform.openai.com/usage`)
	}

	// attempt to parse the structured JSON error from the response body
	var errorResp common.ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		// FALLBACK if the response was not the expected JSON format:
		return fmt.Errorf("OpenAI API request failed with status %d: %s", statusCode, string(body))
	}

	// now we can return a much more helpful, specific error message!
	if errorResp.Error.Message != "" {
		return fmt.Errorf("OpenAI API error: %s", errorResp.Error.Message)
	}

	// final catch-all if parsing succeeded but the message was empty
	return fmt.Errorf("an unknown API error occurred (status %d)", statusCode)
}

// HandleConnectionError handles connection failures - for cloud services, return original error
func (p *Provider) HandleConnectionError(err error) error {
	return err
}

// CustomizeRequest allows OpenAI to customize the HTTP request if needed
func (p *Provider) CustomizeRequest(req *http.Request) error {
	// no customization needed at this time
	// this is implemented for completeness/future extensibility
	return nil
}
