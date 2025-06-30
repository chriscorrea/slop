package groq

import "slop/internal/llm/common"

// ChatRequest represents the request payload for Groq's chat API
// Groq uses OpenAI-compatible format
type ChatRequest struct {
	Model            string                 `json:"model"`
	Messages         []common.Message       `json:"messages"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Stream           *bool                  `json:"stream,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	ResponseFormat   *common.ResponseFormat `json:"response_format,omitempty"`
	Seed             *int                   `json:"seed,omitempty"`
}
