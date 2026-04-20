package anthropic

import (
	"encoding/json"

	"github.com/chriscorrea/slop/internal/llm/common"
)

// MessagesRequest represents the request payload for Anthropic's Messages API
type MessagesRequest struct {
	Model         string           `json:"model"`
	MaxTokens     int              `json:"max_tokens"`
	Messages      []common.Message `json:"messages"`
	System        string           `json:"system,omitempty"`
	Temperature   *float64         `json:"temperature,omitempty"`
	TopP          *float64         `json:"top_p,omitempty"`
	TopK          *int             `json:"top_k,omitempty"`
	StopSequences []string         `json:"stop_sequences,omitempty"`
	Stream        *bool            `json:"stream,omitempty"`

	// Thinking carries the extended thinking config. When nil, the field
	// is omitted and the model behaves as normal
	Thinking *ThinkingConfig `json:"thinking,omitempty"`

	// OutputConfig carries the structured-output envelope. Anthropic
	// wraps the json_schema response format under output_config.format
	OutputConfig *OutputConfig `json:"output_config,omitempty"`
}

// ThinkingConfig wires Anthropic's extended-thinking block. Type is
// "enabled" with a BudgetTokens ceiling on 4.5 and earlier; Type is
// "adaptive" (no budget) on 4.6+. Effort for adaptive routing lives on
// OutputConfig, not here
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// OutputConfig wraps Anthropic's structured-output envelope on the
// Messages API. Format carries the schema and its metadata; Effort
// controls token spend on models that support it (low/medium/high/max).
// Both fields may be set together
type OutputConfig struct {
	Format *OutputFormat `json:"format,omitempty"`
	Effort string        `json:"effort,omitempty"`
}

// OutputFormat describes a single structured-output format. For
// json_schema requests, Name, Schema, and Strict are set
type OutputFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name,omitempty"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict *bool           `json:"strict,omitempty"`
}

// MessagesResponse represents Anthropic's Messages API response
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentItem  `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence"`
	Usage        AnthropicUsage `json:"usage"`
}

// ContentItem represents a content item in Anthropic's response.
// Text carries the body of a "text" block; Thinking carries the body
// of a "thinking" block emitted when extended thinking is enabled
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

// AnthropicUsage represents usage information in Anthropic's format
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
