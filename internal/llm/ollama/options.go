package ollama

import "github.com/chriscorrea/slop/internal/llm/common"

// GenerateOptions contains Ollama-specific generation parameters
type GenerateOptions struct {
	common.GenerateOptions

	// Ollama-specific parameters
	TopK          *int     // Limits token selection to top K candidates
	RepeatPenalty *float64 // Penalty for repeating tokens (default: 1.1)
	Seed          *int     // Random seed for deterministic generation
	Think         *bool    // native think flag for structured reasoning
	KeepAlive     *string  // how long Ollama keeps the model warm in RAM
}

// GenerateOption configures Ollama-specific generation parameters
type GenerateOption func(*GenerateOptions)

// NewGenerateOptions creates new GenerateOptions with functional options applied
func NewGenerateOptions(opts ...GenerateOption) *GenerateOptions {
	config := &GenerateOptions{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// Ollama-specific option functions

// WithTopK limits token selection to top K candidates
func WithTopK(topK int) GenerateOption {
	return func(c *GenerateOptions) {
		c.TopK = &topK
	}
}

// WithRepeatPenalty sets penalty for repeating tokens (default: 1.1)
func WithRepeatPenalty(penalty float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.RepeatPenalty = &penalty
	}
}

// WithSeed enables deterministic generation
func WithSeed(seed int) GenerateOption {
	return func(c *GenerateOptions) {
		c.Seed = &seed
	}
}

// WithThink sets Ollama's native think flag. When modern models
// return thinking content in a separate message.thinking field
// the adapter routes to Message.Thinking
func WithThink(think bool) GenerateOption {
	return func(c *GenerateOptions) {
		c.Think = &think
	}
}

// WithKeepAlive tunes how long Ollama keeps the model warm in RAM after a
// request. Accepts Go-style durations ("5m", "1h") or "0" to unload.
func WithKeepAlive(s string) GenerateOption {
	return func(c *GenerateOptions) {
		c.KeepAlive = &s
	}
}

// Common options - lightweight wrappers that delegate to common functions

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
