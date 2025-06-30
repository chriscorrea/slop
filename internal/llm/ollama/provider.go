package ollama

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"slop/internal/config"
	"slop/internal/llm/common"
)

// Provider implements the unified registry.Provider interface for Ollama
type Provider struct{}

// ensure Provider implements the common.Provider interface
var _ common.Provider = (*Provider)(nil)

// New creates a new Ollama provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	// create client options
	var opts []common.ClientOption
	if cfg.Providers.Ollama.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.Ollama.BaseUrl))
	} else {
		opts = append(opts, common.WithBaseURL("http://localhost:11434"))
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

	// Ollama runs locally and doesn't require an API key
	adapterClient := common.NewAdapterClient(p, "", "http://localhost:11434", opts...)
	return adapterClient, nil
}

// BuildOptions creates Ollama-specific generation options from configuration
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

// RequiresAPIKey returns false (Ollama doesn't require an API key)
func (p *Provider) RequiresAPIKey() bool {
	return false
}

// ProviderName returns the name of this provider
func (p *Provider) ProviderName() string {
	return "ollama"
}

// BuildRequest creates an Ollama-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// convert options to Ollama-specific options
	var config *GenerateOptions
	if options != nil {
		if ollamaOpts, ok := options.(*GenerateOptions); ok {
			config = ollamaOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request using common utilities
	common.LogAPIRequest(logger, "Ollama", modelName, messages, &config.GenerateOptions)

	// create Ollama-specific request payload
	requestBody := &ChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   false, // disable streaming for now
	}

	// build options map for Ollama-specific parameters
	optionsMap := make(map[string]interface{})

	// map common generation options to Ollama's options format
	if config.Temperature != nil {
		optionsMap["temperature"] = *config.Temperature
	}
	if config.TopP != nil {
		optionsMap["top_p"] = *config.TopP
	}
	if config.MaxTokens != nil {
		optionsMap["num_predict"] = *config.MaxTokens // Ollama uses num_predict
	}
	if len(config.Stop) > 0 {
		optionsMap["stop"] = config.Stop
	}

	// map Ollama-specific options
	if config.TopK != nil {
		optionsMap["top_k"] = *config.TopK
	}
	if config.RepeatPenalty != nil {
		optionsMap["repeat_penalty"] = *config.RepeatPenalty
	}
	if config.Seed != nil {
		optionsMap["seed"] = *config.Seed
	}

	// only set options if we have any
	if len(optionsMap) > 0 {
		requestBody.Options = optionsMap
	}

	// handle structured output if requested
	if config.ResponseFormat != nil && config.ResponseFormat.Type == "json_object" {
		requestBody.Format = "json"
	}

	return requestBody, nil
}

// ParseResponse parses an Ollama API response and extracts content and usage
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// parse the Ollama-specific response format
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal Ollama response: %w", err)
	}

	// check if response is complete
	if !chatResp.Done {
		return "", nil, fmt.Errorf("incomplete response received from Ollama (done: false)")
	}

	// log token usage if available
	var usage *common.Usage
	if chatResp.Done && chatResp.PromptEvalCount > 0 {
		usage = &common.Usage{
			PromptTokens:     chatResp.PromptEvalCount,
			CompletionTokens: chatResp.EvalCount,
			TotalTokens:      chatResp.PromptEvalCount + chatResp.EvalCount,
		}
	}

	content := chatResp.Message.Content

	// return content and usage information
	return content, usage, nil
}

// HandleError creates Ollama-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// error message due to invalid model name or missing model
	if strings.Contains(string(body), "try pulling") || strings.Contains(string(body), "not found") {
		return fmt.Errorf(`The requested model was not found. It may not be installed locally or available on the Ollama server.

To see all models you have installed, run:
    ollama list

To download a model, use:
    ollama pull [model_name]

To see available models, visit: https://ollama.com/search`)
	}

	// 413 status code / request is too large
	if statusCode == http.StatusRequestEntityTooLarge {
		return fmt.Errorf(`the request was too large for Ollama to process. 

Please reduce the size of your input or select a model with a larger context window.`)
	}

	// final catch-all
	if len(body) > 0 {
		return fmt.Errorf("an ollama API error occurred (status %d): %s", statusCode, string(body))
	}
	return fmt.Errorf("an ollama API error occurred: status %d", statusCode)

}

// HandleConnectionError provides helpful guidance when Ollama is unreachable
func (p *Provider) HandleConnectionError(err error) error {
	errStr := err.Error()

	// check if this is a connection-related error
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "connect: connection refused") {
		return fmt.Errorf(`Cannot connect to Ollama server.

Ollama may not be running or installed:
• Install Ollama: https://ollama.ai/
• Start Ollama: ollama serve
• Check if Ollama is running: curl http://localhost:11434/api/version

Original error: %s`, errStr)
	}

	// for other types of errors, return the original error
	return err
}

// CustomizeRequest customzise requset for Ollama
func (p *Provider) CustomizeRequest(req *http.Request) error {
	// Ollama uses /api/chat endpoint instead of /chat/completions
	if strings.HasSuffix(req.URL.Path, "/chat/completions") {
		req.URL.Path = strings.Replace(req.URL.Path, "/chat/completions", "/api/chat", 1)
	}

	// note that Ollama doesn't require authentication headers/content-type is already set

	return nil
}
