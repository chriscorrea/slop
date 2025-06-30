package cohere

import "slop/internal/llm/common"

// GenerateOptions contains Cohere-specific generation parameters
type GenerateOptions struct {
	common.GenerateOptions

	// Cohere-specific parameters
	TopK       *int    // limits token selection top K candidates
	Seed       *int    // random seed for deterministic generation
	SafetyMode *string // controls safety instructions ("STRICT", "CONTEXTUAL", "NONE")
}

// GenerateOption configures Cohere-specific generation parameters
type GenerateOption func(*GenerateOptions)

// NewGenerateOptions creates new GenerateOptions with functional options applied
func NewGenerateOptions(opts ...GenerateOption) *GenerateOptions {
	config := &GenerateOptions{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// Cohere-specific option functions

// WithTopK limits token selection to top K candidates
func WithTopK(topK int) GenerateOption {
	return func(c *GenerateOptions) {
		c.TopK = &topK
	}
}

// WithSeed enables deterministic generation
func WithSeed(seed int) GenerateOption {
	return func(c *GenerateOptions) {
		c.Seed = &seed
	}
}

// WithSafetyMode sets the safety mode
// valid values include "STRICT", "CONTEXTUAL", "NONE"
func WithSafetyMode(mode string) GenerateOption {
	return func(c *GenerateOptions) {
		c.SafetyMode = &mode
	}
}

// Common options: lightweight wrappers that delegate to common functions

// WithTemperature sets response randomness (0.0-1.0)
func WithTemperature(temp float64) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithTemperature(temp)(&c.GenerateOptions)
	}
}

// WithTopP sets nucleus sampling threshold (0.0-1.0)
// Note: Cohere uses 'p' parameter instead of 'top_p'
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
