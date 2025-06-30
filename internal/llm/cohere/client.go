package cohere

import "slop/internal/llm/common"

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

	// Cohere-specific parameters
	SafetyMode *string `json:"safety_mode,omitempty"`
}

// ResponseFormat represents Cohere's response format configuration
type ResponseFormat struct {
	Type   string      `json:"type"`
	Schema interface{} `json:"schema,omitempty"`
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
