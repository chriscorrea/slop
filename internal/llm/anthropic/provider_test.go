package anthropic

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"slop/internal/config"
	"slop/internal/llm/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Interface(t *testing.T) {
	// verify that Provider implements the common.Provider interface
	var _ common.Provider = (*Provider)(nil)
}

func TestNew(t *testing.T) {
	provider := New()
	assert.NotNil(t, provider)
	assert.IsType(t, &Provider{}, provider)
}

func TestProvider_ProviderName(t *testing.T) {
	provider := New()
	assert.Equal(t, "anthropic", provider.ProviderName())
}

func TestProvider_RequiresAPIKey(t *testing.T) {
	provider := New()
	assert.True(t, provider.RequiresAPIKey())
}

func TestProvider_CreateClient(t *testing.T) {
	provider := New()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "missing API key",
			config: &config.Config{
				Providers: config.Providers{
					Anthropic: config.Anthropic{},
				},
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Providers: config.Providers{
					Anthropic: config.Anthropic{
						BaseProvider: config.BaseProvider{
							APIKey: "test-api-key",
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
			name: "config with custom base URL",
			config: &config.Config{
				Providers: config.Providers{
					Anthropic: config.Anthropic{
						BaseProvider: config.BaseProvider{
							APIKey:  "test-api-key",
							BaseUrl: "https://custom.anthropic.com",
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := provider.CreateClient(tt.config, slog.Default())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestProvider_BuildOptions(t *testing.T) {
	provider := New()

	tests := []struct {
		name     string
		config   *config.Config
		expected int // expected number of options
	}{
		{
			name: "minimal config",
			config: &config.Config{
				Parameters: config.Parameters{},
				Format:     config.Format{},
			},
			expected: 1, // always returns at least one GenerateOptions object
		},
		{
			name: "full config",
			config: &config.Config{
				Parameters: config.Parameters{
					SystemPrompt:  "You are a helpful assistant.",
					Temperature:   0.7,
					MaxTokens:     1000,
					TopP:          0.9,
					StopSequences: []string{"end", "stop"},
				},
				Format: config.Format{
					JSON: true,
				},
			},
			expected: 1, // still returns one GenerateOptions object, but with all options set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := provider.BuildOptions(tt.config)
			assert.Len(t, options, tt.expected)

			// verify the first option is a GenerateOptions
			if len(options) > 0 {
				generateOpts, ok := options[0].(*GenerateOptions)
				assert.True(t, ok)
				assert.NotNil(t, generateOpts)
			}
		})
	}
}

func TestProvider_BuildRequest(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Can you not understand that liberty is worth more than just ribbons?"},
	}
	modelName := "claude-3-5-sonnet-latest"

	tests := []struct {
		name     string
		options  interface{}
		expected *MessagesRequest
	}{
		{
			name:    "nil options",
			options: nil,
			expected: &MessagesRequest{
				Model:     modelName,
				Messages:  []common.Message{{Role: "user", Content: "Can you not understand that liberty is worth more than just ribbons?"}},
				System:    "You are a helpful assistant.",
				MaxTokens: 1024,
				Stream:    common.BoolPtr(false),
			},
		},
		{
			name: "with generate options",
			options: NewGenerateOptions(
				WithTemperature(0.7),
				WithMaxTokens(1500),
				WithTopK(10),
				WithStopSequences([]string{"end", "stop"}),
			),
			expected: &MessagesRequest{
				Model:         modelName,
				Messages:      []common.Message{{Role: "user", Content: "Can you not understand that liberty is worth more than just ribbons?"}},
				System:        "You are a helpful assistant.",
				Temperature:   common.Float64Ptr(0.7),
				MaxTokens:     1500,
				TopK:          common.IntPtr(10),
				StopSequences: []string{"end", "stop"},
				Stream:        common.BoolPtr(false),
			},
		},
		{
			name:    "invalid options type",
			options: "invalid",
			expected: &MessagesRequest{
				Model:     modelName,
				Messages:  []common.Message{{Role: "user", Content: "Can you not understand that liberty is worth more than just ribbons?"}},
				System:    "You are a helpful assistant.",
				MaxTokens: 1024,
				Stream:    common.BoolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := provider.BuildRequest(messages, modelName, tt.options, slog.Default())

			assert.NoError(t, err)
			assert.NotNil(t, request)

			msgReq, ok := request.(*MessagesRequest)
			require.True(t, ok)

			assert.Equal(t, tt.expected.Model, msgReq.Model)
			assert.Equal(t, tt.expected.Messages, msgReq.Messages)
			assert.Equal(t, tt.expected.System, msgReq.System)
			assert.Equal(t, tt.expected.Stream, msgReq.Stream)

			if tt.expected.Temperature != nil {
				require.NotNil(t, msgReq.Temperature)
				assert.Equal(t, *tt.expected.Temperature, *msgReq.Temperature)
			}
			if tt.expected.MaxTokens != 0 {
				assert.Equal(t, tt.expected.MaxTokens, msgReq.MaxTokens)
			}
			if tt.expected.TopK != nil {
				require.NotNil(t, msgReq.TopK)
				assert.Equal(t, *tt.expected.TopK, *msgReq.TopK)
			}
			if len(tt.expected.StopSequences) > 0 {
				assert.Equal(t, tt.expected.StopSequences, msgReq.StopSequences)
			}
		})
	}
}

func TestProvider_BuildRequest_SystemMessageHandling(t *testing.T) {
	provider := New()

	tests := []struct {
		name           string
		messages       []common.Message
		expectedSystem string
		expectedMsgLen int
	}{
		{
			name: "single system message",
			messages: []common.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hello"},
			},
			expectedSystem: "You are helpful.",
			expectedMsgLen: 1,
		},
		{
			name: "multiple system messages",
			messages: []common.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hello"},
				{Role: "system", Content: "Be concise."},
			},
			expectedSystem: "You are helpful.\n\nBe concise.",
			expectedMsgLen: 1,
		},
		{
			name: "no system messages",
			messages: []common.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			expectedSystem: "",
			expectedMsgLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := provider.BuildRequest(tt.messages, "claude-3-5-sonnet-latest", nil, slog.Default())

			assert.NoError(t, err)
			require.NotNil(t, request)

			msgReq, ok := request.(*MessagesRequest)
			require.True(t, ok)

			assert.Equal(t, tt.expectedSystem, msgReq.System)
			assert.Len(t, msgReq.Messages, tt.expectedMsgLen)
		})
	}
}

func TestProvider_ParseResponse(t *testing.T) {
	provider := New()

	tests := []struct {
		name            string
		responseBody    string
		expectedError   bool
		expectedUsage   *common.Usage
		expectedContent string
	}{
		{
			name: "valid response",
			responseBody: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [
					{
						"type": "text",
						"text": "Hello! I'm doing well, thank you for asking."
					}
				],
				"model": "claude-3-5-sonnet-latest",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 10,
					"output_tokens": 15
				}
			}`,
			expectedError: false,
			expectedUsage: &common.Usage{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
			expectedContent: "Hello! I'm doing well, thank you for asking.",
		},
		{
			name: "multiple content items",
			responseBody: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [
					{
						"type": "text",
						"text": "Hello! "
					},
					{
						"type": "text",
						"text": "How are you?"
					}
				],
				"model": "claude-3-5-sonnet-latest",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 5,
					"output_tokens": 8
				}
			}`,
			expectedError:   false,
			expectedContent: "Hello! How are you?",
		},
		{
			name:          "invalid JSON",
			responseBody:  `{invalid json}`,
			expectedError: true,
		},
		{
			name: "no content",
			responseBody: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [],
				"model": "claude-3-5-sonnet-latest",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 10,
					"output_tokens": 0
				}
			}`,
			expectedError: true,
		},
		{
			name: "no text content",
			responseBody: `{
				"id": "msg_123",
				"type": "message", 
				"role": "assistant",
				"content": [
					{
						"type": "image",
						"source": {}
					}
				],
				"model": "claude-3-5-sonnet-latest",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 10,
					"output_tokens": 0
				}
			}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, usage, err := provider.ParseResponse([]byte(tt.responseBody), slog.Default())

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedContent, content)
				if tt.expectedUsage != nil {
					require.NotNil(t, usage)
					assert.Equal(t, tt.expectedUsage.PromptTokens, usage.PromptTokens)
					assert.Equal(t, tt.expectedUsage.CompletionTokens, usage.CompletionTokens)
					assert.Equal(t, tt.expectedUsage.TotalTokens, usage.TotalTokens)
				}
			}
		})
	}
}

func TestProvider_HandleError(t *testing.T) {
	provider := New()

	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedSubstr string
	}{
		{
			name:           "unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   "",
			expectedSubstr: "Anthropic API authentication failed",
		},
		{
			name:           "rate limit exceeded",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   "",
			expectedSubstr: "Anthropic API rate limit exceeded",
		},
		{
			name:           "structured error response",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"type": "error", "error": {"type": "invalid_request_error", "message": "Invalid request format"}}`,
			expectedSubstr: "Anthropic API error: Invalid request format",
		},
		{
			name:           "malformed error response",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "Internal server error",
			expectedSubstr: "Anthropic API request failed with status 500",
		},
		{
			name:           "empty error message",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"type": "error", "error": {"type": "invalid_request_error", "message": ""}}`,
			expectedSubstr: "an unknown API error occurred (status 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.HandleError(tt.statusCode, []byte(tt.responseBody))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedSubstr)
		})
	}
}

func TestProvider_HandleConnectionError(t *testing.T) {
	provider := New()
	originalErr := assert.AnError

	err := provider.HandleConnectionError(originalErr)
	assert.Equal(t, originalErr, err)
}

func TestProvider_CustomizeRequest(t *testing.T) {
	provider := New()

	tests := []struct {
		name         string
		originalPath string
		expectedPath string
	}{
		{
			name:         "chat completions endpoint",
			originalPath: "/chat/completions",
			expectedPath: "/v1/messages",
		},
		{
			name:         "v1 chat completions endpoint",
			originalPath: "/v1/chat/completions",
			expectedPath: "/v1/messages",
		},
		{
			name:         "already correct endpoint",
			originalPath: "/v1/messages",
			expectedPath: "/v1/messages",
		},
		{
			name:         "other endpoint",
			originalPath: "/other/endpoint",
			expectedPath: "/other/endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "https://api.anthropic.com"+tt.originalPath, nil)

			err := provider.CustomizeRequest(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPath, req.URL.Path)
			assert.Equal(t, "2023-06-01", req.Header.Get("anthropic-version"))
		})
	}
}

func TestProvider_CustomizeRequest_AuthHeaders(t *testing.T) {
	provider := New()

	tests := []struct {
		name               string
		authHeader         string
		expectedXAPIKey    string
		expectedAuthHeader string
	}{
		{
			name:               "Bearer token conversion",
			authHeader:         "Bearer sk-ant-test123",
			expectedXAPIKey:    "sk-ant-test123",
			expectedAuthHeader: "",
		},
		{
			name:               "No auth header",
			authHeader:         "",
			expectedXAPIKey:    "",
			expectedAuthHeader: "",
		},
		{
			name:               "Invalid auth header format",
			authHeader:         "InvalidFormat test123",
			expectedXAPIKey:    "",
			expectedAuthHeader: "InvalidFormat test123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			err := provider.CustomizeRequest(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedXAPIKey, req.Header.Get("x-api-key"))
			assert.Equal(t, tt.expectedAuthHeader, req.Header.Get("Authorization"))
			assert.Equal(t, "2023-06-01", req.Header.Get("anthropic-version"))
		})
	}
}

// Integration test with mock HTTP server
func TestProvider_Integration(t *testing.T) {
	// create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "", r.Header.Get("Authorization")) // Should be removed
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		// parse request body
		var reqBody MessagesRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, "claude-3-5-sonnet-latest", reqBody.Model)
		assert.Len(t, reqBody.Messages, 1)
		assert.Equal(t, "user", reqBody.Messages[0].Role)
		assert.Equal(t, "test message", reqBody.Messages[0].Content)

		// send response
		response := MessagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []ContentItem{
				{
					Type: "text",
					Text: "test response",
				},
			},
			Model:      "claude-3-5-sonnet-latest",
			StopReason: "end_turn",
			Usage: AnthropicUsage{
				InputTokens:  5,
				OutputTokens: 2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// create provider with custom base URL
	provider := New()
	cfg := &config.Config{
		Providers: config.Providers{
			Anthropic: config.Anthropic{
				BaseProvider: config.BaseProvider{
					APIKey:  "test-api-key",
					BaseUrl: server.URL,
				},
			},
		},
	}

	// create client
	client, err := provider.CreateClient(cfg, slog.Default())
	require.NoError(t, err)

	// test generation
	messages := []common.Message{
		{Role: "user", Content: "test message"},
	}

	content, err := client.Generate(context.Background(), messages, "claude-3-5-sonnet-latest")
	require.NoError(t, err)
	assert.Equal(t, "test response", content)
}
