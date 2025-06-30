package mistral

import "slop/internal/llm/common"

// ChatRequest represents the request payload for Mistral's chat API
// This is used by the MistralAdapter to build provider-specific requests
type ChatRequest struct {
	Model          string                 `json:"model"`
	Messages       []common.Message       `json:"messages"`
	Temperature    *float64               `json:"temperature,omitempty"`
	TopP           *float64               `json:"top_p,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	Stream         *bool                  `json:"stream,omitempty"`
	Stop           []string               `json:"stop,omitempty"`
	RandomSeed     *int                   `json:"random_seed,omitempty"`
	ResponseFormat *common.ResponseFormat `json:"response_format,omitempty"`
}
