package mistral

import (
	"encoding/json"

	"github.com/chriscorrea/slop/internal/llm/common"
)

// ChatRequest represents the request payload for Mistral's chat API
// This is used by the MistralAdapter to build provider-specific requests
type ChatRequest struct {
	Model           string                  `json:"model"`
	Messages        []common.Message        `json:"messages"`
	Temperature     *float64                `json:"temperature,omitempty"`
	TopP            *float64                `json:"top_p,omitempty"`
	MaxTokens       *int                    `json:"max_tokens,omitempty"`
	Stream          *bool                   `json:"stream,omitempty"`
	Stop            []string                `json:"stop,omitempty"`
	RandomSeed      *int                    `json:"random_seed,omitempty"`
	ResponseFormat  *ResponseFormatEnvelope `json:"response_format,omitempty"`
	ReasoningEffort *string                 `json:"reasoning_effort,omitempty"`
}

// ResponseFormatEnvelope is Mistral's OpenAI-compatible response_format payload.
// For plain JSON mode, only Type is set (e.g. "json_object"). For schema-
// constrained output, Type is "json_schema" and JSONSchema carries the
// nested envelope {name, schema, strict}.
type ResponseFormatEnvelope struct {
	Type       string          `json:"type"`
	JSONSchema *JSONSchemaSpec `json:"json_schema,omitempty"`
}

// JSONSchemaSpec is the nested payload inside response_format when schema-
// constrained output is requested.
type JSONSchemaSpec struct {
	Name   string          `json:"name,omitempty"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict *bool           `json:"strict,omitempty"`
}
