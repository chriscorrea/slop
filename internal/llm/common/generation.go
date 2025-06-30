package common

// GenerateOptions contains near-universal generation parameters
// this follows the interface segregation principle; providers only see relevant options
type GenerateOptions struct {
	// Core generation parameters are widely supported across providers
	Temperature *float64 // temp parameter in responses (0.0-2.0)
	TopP        *float64 // nucleus sampling parameter (0.0-1.0)
	MaxTokens   *int     // maximum tokens to generate
	Stop        []string // stop sequences

	// structured output - growing adoption across providers
	ResponseFormat *ResponseFormat // structured output format configuration

	// function calling - expect to become standard across providers
	Tools      []ToolConfig // available tools/functions
	ToolChoice interface{}  // tool selection strategy
}

// ToolConfig represents a tool/function definition for function calling
type ToolConfig struct {
	Type     string      `json:"type"`     // e.g., "function"
	Function interface{} `json:"function"` // Function definition
}

// GenerateOption configures generation parameters using the functional options pattern
type GenerateOption func(*GenerateOptions)

// NewGenerateOptions creates a new GenerateOptions with functional options applied
func NewGenerateOptions(opts ...GenerateOption) *GenerateOptions {
	config := &GenerateOptions{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// Core generation options - widely supported across providers

// WithTemperature sets response randomness
// supported by: all providers
func WithTemperature(temp float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.Temperature = &temp
	}
}

// WithTopP sets nucleus sampling threshold (0.0-1.0)
// supported by: most providers
func WithTopP(topP float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.TopP = &topP
	}
}

// WithMaxTokens sets maximum tokens to generate
// supported by: all providers
func WithMaxTokens(maxTokens int) GenerateOption {
	return func(c *GenerateOptions) {
		c.MaxTokens = &maxTokens
	}
}

// WithStop sets stop sequences to halt generation
// supported by: most providers
func WithStop(stop []string) GenerateOption {
	return func(c *GenerateOptions) {
		c.Stop = stop
	}
}

// Structured output options

// WithResponseFormat sets structured output format
// supported by: openai, mistral (growing adoption)
func WithResponseFormat(format *ResponseFormat) GenerateOption {
	return func(c *GenerateOptions) {
		c.ResponseFormat = format
	}
}

// WithJSONFormat enables JSON structured output
// supported by: openai, mistral (growing adoption)
func WithJSONFormat() GenerateOption {
	return func(c *GenerateOptions) {
		c.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}
}

// Function calling options

// WithTools sets available tools/functions for function calling
// widespread adoption expected soon
func WithTools(tools []ToolConfig) GenerateOption {
	return func(c *GenerateOptions) {
		c.Tools = tools
	}
}

// WithToolChoice sets tool selection strategy
// widespread adoption expected soon
func WithToolChoice(choice interface{}) GenerateOption {
	return func(c *GenerateOptions) {
		c.ToolChoice = choice
	}
}
