package ollama

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"slop/internal/config"
	"slop/internal/llm/common"

	"github.com/stretchr/testify/assert"
)

func TestCreateClient(t *testing.T) {
	provider := New()
	logger := slog.Default()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "Success with default config",
			config: &config.Config{
				Providers: config.Providers{
					Ollama: config.Ollama{
						BaseProvider: config.BaseProvider{
							BaseUrl: "http://localhost:11434",
						},
					},
				},
				Parameters: config.Parameters{
					MaxRetries: 3,
				},
			},
			expectError: false,
		},
		{
			name: "Success with custom base URL",
			config: &config.Config{
				Providers: config.Providers{
					Ollama: config.Ollama{
						BaseProvider: config.BaseProvider{
							BaseUrl: "http://custom-ollama:8080",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Success with empty API key (Ollama doesn't require it)",
			config: &config.Config{
				Providers: config.Providers{
					Ollama: config.Ollama{
						BaseProvider: config.BaseProvider{
							APIKey:  "",
							BaseUrl: "http://localhost:11434",
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := provider.CreateClient(tt.config, logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Implements(t, (*common.LLM)(nil), client)
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "user", Content: "Oink"},
	}
	modelName := "gemma3n:latest"

	tests := []struct {
		name     string
		options  interface{}
		validate func(t *testing.T, request interface{})
	}{
		{
			name:    "Build request with nil options",
			options: nil,
			validate: func(t *testing.T, request interface{}) {
				chatReq, ok := request.(*ChatRequest)
				assert.True(t, ok, "Request should be *ChatRequest")
				assert.Equal(t, modelName, chatReq.Model)
				assert.Equal(t, messages, chatReq.Messages)
				assert.Equal(t, false, chatReq.Stream)
			},
		},
		{
			name: "Build request with generation options",
			options: &GenerateOptions{
				GenerateOptions: common.GenerateOptions{
					Temperature: common.Float64Ptr(0.8),
					MaxTokens:   common.IntPtr(1000),
					TopP:        common.Float64Ptr(0.9),
					Stop:        []string{"STOP"},
				},
				TopK: common.IntPtr(50),
				Seed: common.IntPtr(42),
			},
			validate: func(t *testing.T, request interface{}) {
				chatReq, ok := request.(*ChatRequest)
				assert.True(t, ok, "Request should be *ChatRequest")
				assert.Equal(t, modelName, chatReq.Model)
				assert.Equal(t, messages, chatReq.Messages)

				// Verify options mapping
				assert.NotNil(t, chatReq.Options)
				options := chatReq.Options
				assert.Equal(t, 0.8, options["temperature"])
				assert.Equal(t, 1000, options["num_predict"]) // MaxTokens -> num_predict
				assert.Equal(t, 0.9, options["top_p"])
				assert.Equal(t, []string{"STOP"}, options["stop"])
				assert.Equal(t, 50, options["top_k"])
				assert.Equal(t, 42, options["seed"])
			},
		},
		{
			name: "Build request with JSON format",
			options: &GenerateOptions{
				GenerateOptions: common.GenerateOptions{
					ResponseFormat: &common.ResponseFormat{Type: "json_object"},
				},
			},
			validate: func(t *testing.T, request interface{}) {
				chatReq, ok := request.(*ChatRequest)
				assert.True(t, ok, "Request should be *ChatRequest")
				assert.Equal(t, "json", chatReq.Format)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := provider.BuildRequest(messages, modelName, tt.options, slog.Default())
			assert.NoError(t, err)
			assert.NotNil(t, request)
			tt.validate(t, request)
		})
	}
}

func TestParseResponse(t *testing.T) {
	provider := New()

	tests := []struct {
		name            string
		responseBody    []byte
		expectError     bool
		expectedContent string
		expectedUsage   *common.Usage
	}{
		{
			name: "Valid response with usage",
			responseBody: []byte(`{
				"message": {
					"content": "Hello! How can I help you today?"
				},
				"prompt_eval_count": 10,
				"eval_count": 15,
				"done": true
			}`),
			expectError:     false,
			expectedContent: "Hello! How can I help you today?",
			expectedUsage: &common.Usage{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		},
		{
			name: "Valid response without usage",
			responseBody: []byte(`{
				"message": {
					"content": "Simple response"
				},
				"done": true
			}`),
			expectError:     false,
			expectedContent: "Simple response",
			expectedUsage:   nil,
		},
		{
			name: "Response not done (streaming incomplete)",
			responseBody: []byte(`{
				"message": {
					"content": "Partial response"
				},
				"done": false
			}`),
			expectError: true,
		},
		{
			name: "Malformed JSON response",
			responseBody: []byte(`{
				"message": {
					"content": "incomplete json"
			}`),
			expectError: true,
		},
		{
			name:         "Empty response",
			responseBody: []byte(``),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, usage, err := provider.ParseResponse(tt.responseBody, slog.Default())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedContent, content)
				if tt.expectedUsage != nil {
					assert.Equal(t, tt.expectedUsage, usage)
				} else {
					assert.Nil(t, usage)
				}
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	provider := New()

	tests := []struct {
		name             string
		statusCode       int
		responseBody     []byte
		expectedContains []string
	}{
		{
			name:             "Not Found error (model not available)",
			statusCode:       http.StatusNotFound,
			responseBody:     []byte(`{"error": "model not found"}`),
			expectedContains: []string{"The requested model was not found", "ollama pull"},
		},
		{
			name:             "Request Entity Too Large error (413)",
			statusCode:       http.StatusRequestEntityTooLarge,
			responseBody:     []byte(`{"error": "request too large"}`),
			expectedContains: []string{"request was too large", "reduce the size", "larger context window"},
		},
		{
			name:             "Bad Request error",
			statusCode:       http.StatusBadRequest,
			responseBody:     []byte(`{"error": "invalid request format"}`),
			expectedContains: []string{"an ollama API error occurred", "invalid request format"},
		},
		{
			name:             "Internal Server Error",
			statusCode:       http.StatusInternalServerError,
			responseBody:     []byte(`{"error": "server error"}`),
			expectedContains: []string{"an ollama API error occurred", "server error"},
		},
		{
			name:             "Malformed error response",
			statusCode:       http.StatusInternalServerError,
			responseBody:     []byte(`invalid json`),
			expectedContains: []string{"an ollama API error occurred", "status 500"},
		},
		{
			name:             "Empty error message in structured response",
			statusCode:       http.StatusBadRequest,
			responseBody:     []byte(`{"error": ""}`),
			expectedContains: []string{"an ollama API error occurred", "status 400"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.HandleError(tt.statusCode, tt.responseBody)
			assert.Error(t, err)

			errorMsg := err.Error()
			for _, expectedText := range tt.expectedContains {
				assert.Contains(t, errorMsg, expectedText)
			}
		})
	}
}

func TestProviderInterface(t *testing.T) {
	provider := New()

	t.Run("RequiresAPIKey", func(t *testing.T) {
		assert.False(t, provider.RequiresAPIKey())
	})

	t.Run("ProviderName", func(t *testing.T) {
		assert.Equal(t, "ollama", provider.ProviderName())
	})

	t.Run("Implements common.Provider interface", func(t *testing.T) {
		assert.Implements(t, (*common.Provider)(nil), provider)
	})

	t.Run("HandleConnectionError", func(t *testing.T) {
		// Test with a connection-related error
		connectionErr := fmt.Errorf("dial tcp 127.0.0.1:11434: connect: connection refused")
		enhancedErr := provider.HandleConnectionError(connectionErr)

		assert.Error(t, enhancedErr)
		assert.Contains(t, enhancedErr.Error(), "Cannot connect to Ollama server")
		assert.Contains(t, enhancedErr.Error(), "ollama serve")

		// Test with a non-connection error (should return original)
		originalErr := assert.AnError
		enhancedErr2 := provider.HandleConnectionError(originalErr)
		assert.Equal(t, originalErr, enhancedErr2)
	})

	t.Run("CustomizeRequest", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "http://localhost:11434/chat/completions", nil)
		err := provider.CustomizeRequest(req)

		assert.NoError(t, err)
		// Ollama uses /api/chat endpoint instead of /chat/completions
		assert.Equal(t, "/api/chat", req.URL.Path)
	})
}
