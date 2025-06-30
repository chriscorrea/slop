package anthropic

import "slop/internal/llm/common"

// GenerateOptions contains Anthropic-specific generation parameters
type GenerateOptions struct {
	common.GenerateOptions

	// Anthropic-specific params
	TopK          *int     // integer for top-k sampling (only used by some Anthropic models)
	System        string   // system prompt for Anthropic (separate from messages)
	StopSequences []string // anthropic uses "stop_sequences" instead of "stop"
}

// GenerateOption configures Anthropic-specific generation parameters
type GenerateOption func(*GenerateOptions)

// NewGenerateOptions creates new GenerateOptions with functional options applied
func NewGenerateOptions(opts ...GenerateOption) *GenerateOptions {
	config := &GenerateOptions{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// WithTopK sets top-k sampling parameter
func WithTopK(topK int) GenerateOption {
	return func(c *GenerateOptions) {
		c.TopK = &topK
	}
}

// WithSystem sets the system prompt for Anthropic
func WithSystem(system string) GenerateOption {
	return func(c *GenerateOptions) {
		c.System = system
	}
}

// WithStopSequences sets stop sequences (Anthropic uses different naming)
func WithStopSequences(sequences []string) GenerateOption {
	return func(c *GenerateOptions) {
		c.StopSequences = sequences
	}
}

// Common options

// WithTemperature sets response randomness (0.0-1.0 for Anthropic)
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

// WithStop sets stop sequences (maps to StopSequences for Anthropic)
func WithStop(stop []string) GenerateOption {
	return func(c *GenerateOptions) {
		c.StopSequences = stop
		// Also set the common stop field for consistency
		common.WithStop(stop)(&c.GenerateOptions)
	}
}

// WithJSONFormat enables JSON structured output (if supported)
func WithJSONFormat() GenerateOption {
	return func(c *GenerateOptions) {
		common.WithJSONFormat()(&c.GenerateOptions)
	}
}

// WithResponseFormat sets structured output format (if supported)
func WithResponseFormat(format *common.ResponseFormat) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithResponseFormat(format)(&c.GenerateOptions)
	}
}

// GetGenerateOptions returns the embedded common GenerateOptions for validation
func (c *GenerateOptions) GetGenerateOptions() *common.GenerateOptions {
	return &c.GenerateOptions
}
