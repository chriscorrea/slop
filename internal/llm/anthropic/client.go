package anthropic

import "slop/internal/llm/common"

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

// ContentItem represents a content item in Anthropic's response
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents usage information in Anthropic's format
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
