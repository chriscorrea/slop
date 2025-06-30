package common

// ChatResponse represents a standard chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice in the response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse represents a standard API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ResponseFormat specifies output format for structured responses
type ResponseFormat struct {
	Type string `json:"type"`
}
