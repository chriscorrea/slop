package together

import "github.com/chriscorrea/slop/internal/llm/common"

// GenerateOptions contains TogetherAI-specific generation parameters
type GenerateOptions struct {
	common.GenerateOptions

	// TogetherAI-specific params
	FrequencyPenalty  *float64
	PresencePenalty   *float64
	RepetitionPenalty *float64
	MinP              *float64
	LogProbs          *bool
	TopLogProbs       *int
	Echo              *bool // Whether to echo the prompt
	N                 *int
	SafetyModel       *string // Safety model to use for content filtering
}

// GenerateOption configures Together.AI-specific generation parameters
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

// WithRepetitionPenalty sets repetition penalty (0.0 to 2.0)
func WithRepetitionPenalty(penalty float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.RepetitionPenalty = &penalty
	}
}

// WithMinP sets minimum probability for token sampling
func WithMinP(minP float64) GenerateOption {
	return func(c *GenerateOptions) {
		c.MinP = &minP
	}
}

// WithLogProbs enables log probabilities in response
func WithLogProbs(logProbs bool) GenerateOption {
	return func(c *GenerateOptions) {
		c.LogProbs = &logProbs
	}
}

// WithTopLogProbs sets number of top log probabilities to return
func WithTopLogProbs(topLogProbs int) GenerateOption {
	return func(c *GenerateOptions) {
		c.TopLogProbs = &topLogProbs
	}
}

// WithEcho enables echoing the prompt in response
func WithEcho(echo bool) GenerateOption {
	return func(c *GenerateOptions) {
		c.Echo = &echo
	}
}

// WithN sets number of completions to generate
func WithN(n int) GenerateOption {
	return func(c *GenerateOptions) {
		c.N = &n
	}
}

// WithSafetyModel sets the safety model for content filtering
func WithSafetyModel(model string) GenerateOption {
	return func(c *GenerateOptions) {
		c.SafetyModel = &model
	}
}

// Common options

// WithTemperature sets response randomness
func WithTemperature(temp float64) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithTemperature(temp)(&c.GenerateOptions)
	}
}

// WithTopP sets nucleus sampling threshold
func WithTopP(topP float64) GenerateOption {
	return func(c *GenerateOptions) {
		common.WithTopP(topP)(&c.GenerateOptions)
	}
}

// WithMaxTokens sets max tokens to generate
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

// WithJSONFormat enables JSON structured output (available for some models)
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
