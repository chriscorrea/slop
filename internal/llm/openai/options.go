package openai

import "slop/internal/llm/common"

// GenerateOptions contains OpenAI-specific generation parameters
type GenerateOptions struct {
	common.GenerateOptions

	// OpenAI-specific params
	FrequencyPenalty *float64    // Number between -2.0 and 2.0
	PresencePenalty  *float64    // Number between -2.0 and 2.0
	Seed             *int        // Integer seed for deterministic outputs
	Tools            []Tool      // Function calling tools
	ToolChoice       interface{} // "none", "auto", or specific tool choice
}

// GenerateOption configures OpenAI-specific generation parameters
type GenerateOption func(*GenerateOptions)

// NewGenerateOptions creates new GenerateOptions with functional options applied
func NewGenerateOptions(opts ...GenerateOption) *GenerateOptions {
	config := &GenerateOptions{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// WithFrequencyPenalty sets frequency penalty (-2.0 to 2.0)
func WithFrequencyPenalty(penalty float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.FrequencyPenalty = &penalty
	}
}

// WithPresencePenalty sets presence penalty (-2.0 to 2.0)
func WithPresencePenalty(penalty float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.PresencePenalty = &penalty
	}
}

// WithSeed enables deterministic generation using seed
func WithSeed(seed int) GenerateOption {
	return func(c *GenerateOptions) {
		c.Seed = &seed
	}
}

// WithTools sets function calling tools
func WithTools(tools []Tool) GenerateOption {
	return func(c *GenerateOptions) {
		c.Tools = tools
	}
}

// WithToolChoice sets tool choice strategy
func WithToolChoice(choice interface{}) GenerateOption {
	return func(c *GenerateOptions) {
		c.ToolChoice = choice
	}
}

// Common options

// WithTemperature sets response randomness (0.0-2.0)
func WithTemperature(temp float64) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithTemperature(temp)(&c.GenerateOptions)
	}
}

// WithTopP sets nucleus sampling threshold (0.0-1.0)
func WithTopP(topP float64) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithTopP(topP)(&c.GenerateOptions)
	}
}

// WithMaxTokens sets maximum tokens to generate
func WithMaxTokens(maxTokens int) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithMaxTokens(maxTokens)(&c.GenerateOptions)
	}
}

// WithStop sets stop sequences to halt generation
func WithStop(stop []string) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithStop(stop)(&c.GenerateOptions)
	}
}

// WithJSONFormat enables JSON structured output
func WithJSONFormat() GenerateOption {
	return func(c *GenerateOptions) {
		common.WithJSONFormat()(&c.GenerateOptions)
	}
}

// WithResponseFormat sets structured output format
func WithResponseFormat(format *common.ResponseFormat) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithResponseFormat(format)(&c.GenerateOptions)
	}
}

// GetGenerateOptions returns the embedded common GenerateOptions for validation
func (c *GenerateOptions) GetGenerateOptions() *common.GenerateOptions {
	return &c.GenerateOptions
}
