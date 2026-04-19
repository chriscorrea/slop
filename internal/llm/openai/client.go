package openai

import (
	"encoding/json"

	"github.com/chriscorrea/slop/internal/llm/common"
)

// ChatRequest represents the request payload for OpenAI's chat API
type ChatRequest struct {
	Model               string              `json:"model"`
	Messages            []common.Message    `json:"messages"`
	Temperature         *float64            `json:"temperature,omitempty"`
	TopP                *float64            `json:"top_p,omitempty"`
	MaxCompletionTokens *int                `json:"max_completion_tokens,omitempty"`
	Stream              *bool               `json:"stream,omitempty"`
	Stop                []string            `json:"stop,omitempty"`
	FrequencyPenalty    *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64            `json:"presence_penalty,omitempty"`
	ResponseFormat      *chatResponseFormat `json:"response_format,omitempty"`
	ReasoningEffort     *string             `json:"reasoning_effort,omitempty"`
	Seed                *int                `json:"seed,omitempty"`
	Tools               []Tool              `json:"tools,omitempty"`
	ToolChoice          interface{}         `json:"tool_choice,omitempty"`
}

// chatResponseFormat is the OpenAI wire-shape for the response_format field.
// For json_object it serializes to {"type":"json_object"}.
// For json_schema it nests the schema under a json_schema envelope.
type chatResponseFormat struct {
	Type       string              `json:"type"`
	JSONSchema *chatJSONSchemaSpec `json:"json_schema,omitempty"`
}

// chatJSONSchemaSpec is the envelope OpenAI expects when using json_schema.
// See https://platform.openai.com/docs/guides/structured-outputs
type chatJSONSchemaSpec struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict *bool           `json:"strict,omitempty"`
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
