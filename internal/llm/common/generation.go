package common

import "fmt"

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

	// Thinking / reasoning effort — translated by each provider adapter into
	// its upstream native parameter
	Thinking ThinkingLevel
}

// ThinkingLevel expresses how much reasoning the model should do. Adapters
// translate this into their provider-specific native parameter.
type ThinkingLevel int

const (
	// ThinkingOff sends no thinking hint and model behaves as normal
	ThinkingOff ThinkingLevel = iota
	// ThinkingMedium asks the model for moderate reasoning effort
	ThinkingMedium
	// ThinkingHigh asks the model for maximum reasoning effort
	ThinkingHigh
)

// ParseThinkingLevel converts a string (e.g. from config or a flag) into a
// ThinkingLevel. Unknown values return an error so callers can surface a
// clear message at config-load time rather than at request time
func ParseThinkingLevel(s string) (ThinkingLevel, error) {
	switch s {
	case "", "off":
		return ThinkingOff, nil
	case "medium":
		return ThinkingMedium, nil
	case "high":
		return ThinkingHigh, nil
	default:
		return ThinkingOff, fmt.Errorf("invalid thinking level %q: expected off|medium|high", s)
	}
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

// WithSchema requests schema-constrained structured output. Adapters that
// support it wrap the schema in their provider-specific envelope
// (e.g.OpenAI json_schema, Anthropic output_config, and so on).
//
// `Strict` defaults to true so adherence is enforced where provider supports it
// Providers that don't understand strict simply ignore the field
func WithSchema(name string, schema []byte) GenerateOption {
	strict := true
	return func(c *GenerateOptions) {
		c.ResponseFormat = &ResponseFormat{
			Type:   "json_schema",
			Name:   name,
			Schema: schema,
			Strict: &strict,
		}
	}
}

// WithThinking sets the requested reasoning effort. Adapters translate this
// into their provider-specific native parameter
// If provider/model doesn't support, silently no-op/ignore the request
func WithThinking(level ThinkingLevel) GenerateOption {
	return func(c *GenerateOptions) {
		c.Thinking = level
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
