package groq

import (
	"encoding/json"

	"github.com/chriscorrea/slop/internal/llm/common"
)

// ChatRequest represents the request payload for Groq's chat API.
// Groq uses an OpenAI-compatible wire format.
type ChatRequest struct {
	Model            string              `json:"model"`
	Messages         []common.Message    `json:"messages"`
	Temperature      *float64            `json:"temperature,omitempty"`
	TopP             *float64            `json:"top_p,omitempty"`
	MaxTokens        *int                `json:"max_tokens,omitempty"`
	Stream           *bool               `json:"stream,omitempty"`
	Stop             []string            `json:"stop,omitempty"`
	FrequencyPenalty *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64            `json:"presence_penalty,omitempty"`
	ResponseFormat   *chatResponseFormat `json:"response_format,omitempty"`
	ReasoningFormat  *string             `json:"reasoning_format,omitempty"`
	Seed             *int                `json:"seed,omitempty"`
}

// chatResponseFormat is Groq's OpenAI-compatible wire shape for the
// response_format field. For json_object it serializes to
// {"type":"json_object"}; for json_schema it nests the schema under a
// json_schema envelope (matching OpenAI's structured-outputs shape).
type chatResponseFormat struct {
	Type       string              `json:"type"`
	JSONSchema *chatJSONSchemaSpec `json:"json_schema,omitempty"`
}

// chatJSONSchemaSpec is the envelope Groq expects under response_format
// when using json_schema. See https://console.groq.com/docs/structured-outputs
type chatJSONSchemaSpec struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict *bool           `json:"strict,omitempty"`
}
