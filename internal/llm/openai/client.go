package openai

import "slop/internal/llm/common"

// ChatRequest represents the request payload for OpenAI's chat API
type ChatRequest struct {
	Model               string                 `json:"model"`
	Messages            []common.Message       `json:"messages"`
	Temperature         *float64               `json:"temperature,omitempty"`
	TopP                *float64               `json:"top_p,omitempty"`
	MaxCompletionTokens *int                   `json:"max_completion_tokens,omitempty"`
	Stream              *bool                  `json:"stream,omitempty"`
	Stop                []string               `json:"stop,omitempty"`
	FrequencyPenalty    *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64               `json:"presence_penalty,omitempty"`
	ResponseFormat      *common.ResponseFormat `json:"response_format,omitempty"`
	Seed                *int                   `json:"seed,omitempty"`
	Tools               []Tool                 `json:"tools,omitempty"`
	ToolChoice          interface{}            `json:"tool_choice,omitempty"`
}

// Tool represents an OpenAI function tool
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents an OpenAI function definition
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}
