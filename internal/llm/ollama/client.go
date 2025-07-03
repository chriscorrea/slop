package ollama

import "github.com/chriscorrea/slop/internal/llm/common"

// ChatRequest represents the request payload for Ollama's chat API
type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []common.Message `json:"messages"`
	Stream   bool             `json:"stream"`

	// Generation parameters
	Options map[string]interface{} `json:"options,omitempty"`

	// Structured output support
	Format string `json:"format,omitempty"`
}

// ChatResponse represents the response from Ollama's chat API
type ChatResponse struct {
	Model     string         `json:"model"`
	CreatedAt string         `json:"created_at"`
	Message   common.Message `json:"message"`
	Done      bool           `json:"done"`

	// Token usage information (when done=true)
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

// ErrorResponse represents an error response from Ollama
type ErrorResponse struct {
	Error string `json:"error"`
}
