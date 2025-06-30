package mistral

import "slop/internal/llm/common"

// GenerateOptions contains Mistral-specific generation parameters
type GenerateOptions struct {
	common.GenerateOptions

	// Mistral-specific params
	RandomSeed *int // mistral uses "random_seed" instead of "seed"
}

// GenerateOption configures Mistral-specific generation parameters
type GenerateOption func(*GenerateOptions)

// NewGenerateOptions creates new GenerateOptions with functional options applied
func NewGenerateOptions(opts ...GenerateOption) *GenerateOptions {
	config := &GenerateOptions{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// WithRandomSeed enables deterministic generation using Mistral's random_seed field
func WithRandomSeed(seed int) GenerateOption {
	return func(c *GenerateOptions) {
		c.RandomSeed = &seed
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
