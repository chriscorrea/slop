package cohere

import (
	"encoding/json"

	"github.com/chriscorrea/slop/internal/llm/common"
)

// ChatRequest represents the request payload for Cohere's chat API
type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []common.Message `json:"messages"`
	Stream   *bool            `json:"stream,omitempty"`

	// Generation parameters
	Temperature   *float64 `json:"temperature,omitempty"`
	MaxTokens     *int     `json:"max_tokens,omitempty"`
	P             *float64 `json:"p,omitempty"` // Cohere uses 'p' instead of 'top_p'
	K             *int     `json:"k,omitempty"` // Cohere's top-k parameter
	Seed          *int     `json:"seed,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`

	// Structured output support
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Tools / function calling
	StrictTools *bool `json:"strict_tools,omitempty"`

	// Grounded generation (RAG). When non-empty, Cohere requires
	// safety_mode to be "CONTEXTUAL".
	Documents []Document `json:"documents,omitempty"`

	// Cohere-specific parameters
	SafetyMode *string `json:"safety_mode,omitempty"`
}

// ResponseFormat represents Cohere's response format configuration.
// Schema is the raw JSON schema body (sent alongside Type=="json_object"
// for schema-constrained output; see Cohere v2 chat docs).
type ResponseFormat struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema,omitempty"`
}

// Document represents a single grounding document for Cohere's RAG flow.
// Data holds arbitrary key/value fields (e.g. title, snippet) that the
// model can cite. ID is optional but useful for citation tracking.
type Document struct {
	ID   string            `json:"id,omitempty"`
	Data map[string]string `json:"data"`
}

// ChatResponse represents the response from Cohere's chat API
type ChatResponse struct {
	ID           string         `json:"id"`
	FinishReason string         `json:"finish_reason"`
	Message      common.Message `json:"message"`
	Usage        Usage          `json:"usage"`
}

// Usage represents token usage information from Cohere
type Usage struct {
	BilledUnits BilledUnits `json:"billed_units"`
	Tokens      Tokens      `json:"tokens"`
}

// BilledUnits represents billed token counts
type BilledUnits struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Tokens represents detailed token counts
type Tokens struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorResponse represents an error response from Cohere
type ErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}
