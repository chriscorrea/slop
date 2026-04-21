package together

import (
	"encoding/json"

	"github.com/chriscorrea/slop/internal/llm/common"
)

// ChatRequest represents the request payload for Together.AI's chat API
// Together.AI uses an OpenAI-compatible format for chat completions
type ChatRequest struct {
	Model             string              `json:"model"`
	Messages          []common.Message    `json:"messages"`
	Temperature       *float64            `json:"temperature,omitempty"`
	MaxTokens         *int                `json:"max_tokens,omitempty"`
	TopP              *float64            `json:"top_p,omitempty"`
	Stop              []string            `json:"stop,omitempty"`
	Stream            *bool               `json:"stream,omitempty"`
	ResponseFormat    *ChatResponseFormat `json:"response_format,omitempty"`
	FrequencyPenalty  *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty   *float64            `json:"presence_penalty,omitempty"`
	RepetitionPenalty *float64            `json:"repetition_penalty,omitempty"`
	LogProbs          *bool               `json:"logprobs,omitempty"`
	TopLogProbs       *int                `json:"top_logprobs,omitempty"`
	Echo              *bool               `json:"echo,omitempty"`
	N                 *int                `json:"n,omitempty"`
	MinP              *float64            `json:"min_p,omitempty"`
	SafetyModel       *string             `json:"safety_model,omitempty"`
}

// ChatResponseFormat is Together.AI's response_format wire shape. Together is
// OpenAI-compatible and supports two variants:
//   - legacy: {"type": "json_object"}
//   - envelope: {"type": "json_schema", "json_schema": {name, schema, strict}}
type ChatResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *JSONSchemaConfig `json:"json_schema,omitempty"`
}

// JSONSchemaConfig is the nested body of the json_schema envelope, mirroring
// OpenAI's structured outputs wire shape.
type JSONSchemaConfig struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict *bool           `json:"strict,omitempty"`
}

// ErrorResponse represents Together.AI's error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error details from Together.AI
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}
