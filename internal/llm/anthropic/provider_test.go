package anthropic

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"

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
				MaxTokens: maxTokensDefault,
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
				MaxTokens: maxTokensDefault,
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

// TestSupportsThinking verifies the model id allowlist that drives
// whether BuildRequest emits Anthropic's extended-thinking block.
func TestSupportsThinking(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{name: "sonnet 4.6 supports", modelID: "claude-sonnet-4-6", want: true},
		{name: "sonnet 4 latest supports", modelID: "claude-sonnet-4-latest", want: true},
		{name: "opus 4.7 supports", modelID: "claude-opus-4-7", want: true},
		{name: "opus 4.0 supports", modelID: "claude-opus-4-0", want: true},
		{name: "3-7 sonnet supports", modelID: "claude-3-7-sonnet-latest", want: true},
		{name: "haiku 3 does not", modelID: "claude-3-haiku-20240307", want: false},
		{name: "haiku 4.5 does not", modelID: "claude-haiku-4-5", want: false},
		{name: "3-5 sonnet does not", modelID: "claude-3-5-sonnet-latest", want: false},
		{name: "empty does not", modelID: "", want: false},
		{name: "unknown does not", modelID: "snowball-1.0", want: false},
		{name: "uppercase opus still supports", modelID: "CLAUDE-OPUS-4-7", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, supportsThinking(tt.modelID))
		})
	}
}

// TestDefaultMaxTokens covers the per-model fallback used when the caller
// hasn't set MaxTokens. Sonnet 4 and Opus 4 families get larger budgets so
// the windmill has room to stand; everything else gets the modest default.
func TestDefaultMaxTokens(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    int
	}{
		{name: "sonnet 4.6", modelID: "claude-sonnet-4-6", want: maxTokensSonnetFamily4},
		{name: "sonnet 4.0", modelID: "claude-sonnet-4-0", want: maxTokensSonnetFamily4},
		{name: "opus 4.7", modelID: "claude-opus-4-7", want: maxTokensOpusFamily4},
		{name: "opus 4.5", modelID: "claude-opus-4-5", want: maxTokensOpusFamily4},
		{name: "haiku 4.5", modelID: "claude-haiku-4-5", want: maxTokensDefault},
		{name: "3-5 sonnet", modelID: "claude-3-5-sonnet-latest", want: maxTokensDefault},
		{name: "unknown", modelID: "napoleon-1", want: maxTokensDefault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, defaultMaxTokens(tt.modelID))
		})
	}
}

// TestBuildRequest_Thinking covers the legacy enabled+budget_tokens path
// used for 4.5 and earlier Claude models. Model ids are pinned to 4.5 so
// the adaptive routing doesn't intercept these cases.
func TestBuildRequest_Thinking(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "user", Content: "Why does Boxer trust Napoleon?"},
	}

	t.Run("high on sonnet 4.5 sets budget 16000 and bumps max_tokens, no effort", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingHigh))
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "enabled", msgReq.Thinking.Type)
		assert.Equal(t, 16000, msgReq.Thinking.BudgetTokens)
		// sonnet 4 default is 16384; budget is 16000 so default already
		// clears the constraint without adjustment
		assert.GreaterOrEqual(t, msgReq.MaxTokens, msgReq.Thinking.BudgetTokens+1)
		// sonnet 4.5 isn't in the effort allowlist
		if msgReq.OutputConfig != nil {
			assert.Empty(t, msgReq.OutputConfig.Effort)
		}
	})

	t.Run("medium on opus 4.5 sets budget 4000 and effort medium", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingMedium))
		req, err := provider.BuildRequest(messages, "claude-opus-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "enabled", msgReq.Thinking.Type)
		assert.Equal(t, 4000, msgReq.Thinking.BudgetTokens)
		// opus 4.5 is in the effort allowlist even though it uses manual thinking
		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "medium", msgReq.OutputConfig.Effort)
	})

	t.Run("high on haiku does not emit thinking", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingHigh))
		req, err := provider.BuildRequest(messages, "claude-haiku-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		assert.Nil(t, msgReq.Thinking)
	})

	t.Run("off omits thinking block on opus 4.5 but still sends low effort", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingOff))
		req, err := provider.BuildRequest(messages, "claude-opus-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		assert.Nil(t, msgReq.Thinking)
		// opus 4.5 is in supportsEffort; off maps to low regardless of thinking
		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "low", msgReq.OutputConfig.Effort)
	})

	t.Run("off on sonnet 4.5 emits neither thinking nor effort", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingOff))
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		assert.Nil(t, msgReq.Thinking)
		// sonnet 4.5 isn't in supportsEffort, so no OutputConfig is created
		if msgReq.OutputConfig != nil {
			assert.Empty(t, msgReq.OutputConfig.Effort)
		}
	})

	t.Run("tight max_tokens bumped above budget on 4.5", func(t *testing.T) {
		opts := NewGenerateOptions(
			WithThinking(common.ThinkingHigh),
			WithMaxTokens(2048),
		)
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, 16000, msgReq.Thinking.BudgetTokens)
		assert.Equal(t, 16000+maxTokensDefault, msgReq.MaxTokens)
	})

	t.Run("custom thinking budget is honored on 4.5", func(t *testing.T) {
		opts := NewGenerateOptions(
			WithThinking(common.ThinkingMedium),
			WithThinkingBudget(8000),
		)
		req, err := provider.BuildRequest(messages, "claude-opus-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, 8000, msgReq.Thinking.BudgetTokens)
	})
}

// TestParseAnthropicVersion covers the version-suffix regex, including the
// distinction between "-4-6" (parseable) and "-4-20250514" (date stamp).
func TestParseAnthropicVersion(t *testing.T) {
	tests := []struct {
		name      string
		modelID   string
		wantMajor int
		wantMinor int
		wantOK    bool
	}{
		{name: "sonnet 4.6", modelID: "claude-sonnet-4-6", wantMajor: 4, wantMinor: 6, wantOK: true},
		{name: "opus 4.7", modelID: "claude-opus-4-7", wantMajor: 4, wantMinor: 7, wantOK: true},
		{name: "sonnet 4.5", modelID: "claude-sonnet-4-5", wantMajor: 4, wantMinor: 5, wantOK: true},
		{name: "sonnet 4.6 with date suffix", modelID: "claude-sonnet-4-6-20260101", wantMajor: 4, wantMinor: 6, wantOK: true},
		{name: "4.0-era date snapshot", modelID: "claude-sonnet-4-20250514", wantOK: false},
		{name: "3-7-sonnet dated", modelID: "claude-3-7-sonnet-20241022", wantMajor: 3, wantMinor: 7, wantOK: true},
		{name: "haiku 4.5", modelID: "claude-haiku-4-5", wantMajor: 4, wantMinor: 5, wantOK: true},
		{name: "non-claude id", modelID: "gpt-5", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, ok := parseAnthropicVersion(tt.modelID)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantMajor, major)
				assert.Equal(t, tt.wantMinor, minor)
			}
		})
	}
}

// TestUseAdaptiveThinking verifies the 4.6+ routing gate.
func TestUseAdaptiveThinking(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{name: "sonnet 4.6", modelID: "claude-sonnet-4-6", want: true},
		{name: "opus 4.7", modelID: "claude-opus-4-7", want: true},
		{name: "sonnet 4.5", modelID: "claude-sonnet-4-5", want: false},
		{name: "opus 4.0", modelID: "claude-opus-4-0", want: false},
		{name: "3-7-sonnet", modelID: "claude-3-7-sonnet-20241022", want: false},
		{name: "haiku 4.5", modelID: "claude-haiku-4-5", want: false},
		{name: "date-snapshot 4.0 era", modelID: "claude-sonnet-4-20250514", want: false},
		{name: "non-claude", modelID: "gpt-5", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, useAdaptiveThinking(tt.modelID))
		})
	}
}

// TestSupportsMaxEffort verifies the allowlist for the max effort tier.
func TestSupportsMaxEffort(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{name: "opus 4.6 max-capable", modelID: "claude-opus-4-6", want: true},
		{name: "opus 4.7 max-capable", modelID: "claude-opus-4-7", want: true},
		{name: "sonnet 4.6 max-capable", modelID: "claude-sonnet-4-6", want: true},
		{name: "mythos preview max-capable", modelID: "claude-mythos-preview", want: true},
		{name: "sonnet 4.7 hypothetical falls back", modelID: "claude-sonnet-4-7", want: false},
		{name: "haiku 4.6 hypothetical falls back", modelID: "claude-haiku-4-6", want: false},
		{name: "sonnet 4.5 not max", modelID: "claude-sonnet-4-5", want: false},
		{name: "non-claude not max", modelID: "gpt-5", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, supportsMaxEffort(tt.modelID))
		})
	}
}

// TestEffortForLevel covers the tri-state effort mapping with and without
// max-effort support.
func TestEffortForLevel(t *testing.T) {
	tests := []struct {
		name  string
		level common.ThinkingLevel
		maxOK bool
		want  string
	}{
		{name: "off regardless of maxOK", level: common.ThinkingOff, maxOK: true, want: "low"},
		{name: "off on non-max", level: common.ThinkingOff, maxOK: false, want: "low"},
		{name: "medium regardless of maxOK", level: common.ThinkingMedium, maxOK: true, want: "medium"},
		{name: "medium on non-max", level: common.ThinkingMedium, maxOK: false, want: "medium"},
		{name: "high upgrades to max when allowed", level: common.ThinkingHigh, maxOK: true, want: "max"},
		{name: "high falls back to high when not allowed", level: common.ThinkingHigh, maxOK: false, want: "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, effortForLevel(tt.level, tt.maxOK))
		})
	}
}

// TestBuildRequest_ThinkingAdaptive covers the 4.6+ adaptive path: the
// thinking block carries type=adaptive plus an effort string (never a
// budget_tokens value), and max_tokens is not bumped because adaptive
// self-manages its reasoning budget.
func TestBuildRequest_ThinkingAdaptive(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "user", Content: "Explain the windmill's collapse."},
	}

	t.Run("high on sonnet 4.6 maps to max on output_config", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingHigh))
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-6", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "adaptive", msgReq.Thinking.Type)
		assert.Equal(t, 0, msgReq.Thinking.BudgetTokens)
		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "max", msgReq.OutputConfig.Effort)
	})

	t.Run("high on opus 4.7 maps to max on output_config", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingHigh))
		req, err := provider.BuildRequest(messages, "claude-opus-4-7", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "adaptive", msgReq.Thinking.Type)
		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "max", msgReq.OutputConfig.Effort)
	})

	t.Run("medium on sonnet 4.6 maps to medium on output_config", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingMedium))
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-6", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "adaptive", msgReq.Thinking.Type)
		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "medium", msgReq.OutputConfig.Effort)
	})

	t.Run("off on sonnet 4.6 still sends adaptive with low effort", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingOff))
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-6", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "adaptive", msgReq.Thinking.Type)
		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "low", msgReq.OutputConfig.Effort)
	})

	t.Run("adaptive does not bump max_tokens on tight setting", func(t *testing.T) {
		opts := NewGenerateOptions(
			WithThinking(common.ThinkingHigh),
			WithMaxTokens(2048),
		)
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-6", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, 2048, msgReq.MaxTokens)
	})

	t.Run("high on adaptive-but-not-max falls back to high", func(t *testing.T) {
		// hypothetical future sonnet past 4.6: adaptive routing kicks in
		// (minor=7 >= 6) but the model is not in the max-effort allowlist.
		// this also exercises the adaptive-thinking-without-effort branch
		// since sonnet-4-7 isn't in supportsEffort either
		opts := NewGenerateOptions(WithThinking(common.ThinkingHigh))
		req, err := provider.BuildRequest(messages, "claude-sonnet-4-7", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "adaptive", msgReq.Thinking.Type)
		// sonnet-4-7 isn't yet in supportsEffort, so no effort is sent
		if msgReq.OutputConfig != nil {
			assert.Empty(t, msgReq.OutputConfig.Effort)
		}
	})
}

// TestBuildRequest_OutputConfig verifies that WithSchema wires the schema
// onto Anthropic's output_config envelope with strict mode preserved.
func TestBuildRequest_OutputConfig(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "user", Content: "Return the quote"},
	}
	schema := []byte(`{"type":"object","properties":{"quote":{"type":"string"}}}`)

	opts := NewGenerateOptions(WithSchema("animal_quote", schema))
	req, err := provider.BuildRequest(messages, "claude-sonnet-4-6", opts, slog.Default())
	require.NoError(t, err)

	msgReq, ok := req.(*MessagesRequest)
	require.True(t, ok)
	require.NotNil(t, msgReq.OutputConfig)
	require.NotNil(t, msgReq.OutputConfig.Format)

	fmt := msgReq.OutputConfig.Format
	assert.Equal(t, "json_schema", fmt.Type)
	assert.Equal(t, "animal_quote", fmt.Name)
	assert.JSONEq(t, string(schema), string(fmt.Schema))
	require.NotNil(t, fmt.Strict)
	assert.True(t, *fmt.Strict)
}

// TestBuildRequest_PerModelMaxTokens covers the per-model defaults picked
// when the caller hasn't set MaxTokens.
func TestBuildRequest_PerModelMaxTokens(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "hello"}}

	tests := []struct {
		name    string
		modelID string
		want    int
	}{
		{name: "sonnet 4.6 default", modelID: "claude-sonnet-4-6", want: maxTokensSonnetFamily4},
		{name: "opus 4.7 default", modelID: "claude-opus-4-7", want: maxTokensOpusFamily4},
		{name: "haiku default", modelID: "claude-haiku-4-5", want: maxTokensDefault},
		{name: "legacy sonnet default", modelID: "claude-3-5-sonnet-latest", want: maxTokensDefault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := provider.BuildRequest(messages, tt.modelID, nil, slog.Default())
			require.NoError(t, err)
			msgReq, ok := req.(*MessagesRequest)
			require.True(t, ok)
			assert.Equal(t, tt.want, msgReq.MaxTokens)
		})
	}

	t.Run("caller MaxTokens wins", func(t *testing.T) {
		opts := NewGenerateOptions(WithMaxTokens(1234))
		req, err := provider.BuildRequest(messages, "claude-opus-4-7", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)
		assert.Equal(t, 1234, msgReq.MaxTokens)
	})
}

// TestParseResponse_ThinkingBlocks covers Anthropic's content-block thinking
// being re-inlined as a <think> prefix so the downstream filter treats it
// the same as any other provider's inline-tag thinking.
func TestParseResponse_ThinkingBlocks(t *testing.T) {
	provider := New()

	body := `{
		"id": "msg_abc",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "thinking", "thinking": "Snowball drew the plans for the windmill."},
			{"type": "text", "text": "Four legs good, two legs bad."}
		],
		"model": "claude-opus-4-7",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 12, "output_tokens": 7}
	}`

	content, usage, err := provider.ParseResponse([]byte(body), slog.Default())
	require.NoError(t, err)
	assert.Equal(t,
		"<think>Snowball drew the plans for the windmill.</think>\nFour legs good, two legs bad.",
		content,
	)
	require.NotNil(t, usage)
	assert.Equal(t, 12, usage.PromptTokens)
	assert.Equal(t, 7, usage.CompletionTokens)
	assert.Equal(t, 19, usage.TotalTokens)
}

// TestParseResponse_MultipleThinkingBlocks confirms that multiple thinking
// blocks are concatenated before being wrapped in a single <think> tag.
func TestParseResponse_MultipleThinkingBlocks(t *testing.T) {
	provider := New()

	body := `{
		"id": "msg_abc",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "thinking", "thinking": "First, consider the cowshed."},
			{"type": "thinking", "thinking": " Then the windmill."},
			{"type": "text", "text": "Boxer will work harder."}
		],
		"model": "claude-opus-4-7",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 1, "output_tokens": 1}
	}`

	content, _, err := provider.ParseResponse([]byte(body), slog.Default())
	require.NoError(t, err)
	assert.Equal(t,
		"<think>First, consider the cowshed. Then the windmill.</think>\nBoxer will work harder.",
		content,
	)
}

// TestParseResponse_NoThinking confirms that plain text responses pass
// through untouched when no thinking blocks are present.
func TestParseResponse_NoThinking(t *testing.T) {
	provider := New()

	body := `{
		"id": "msg_abc",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "text", "text": "All animals are equal."}
		],
		"model": "claude-opus-4-7",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 3, "output_tokens": 5}
	}`

	content, _, err := provider.ParseResponse([]byte(body), slog.Default())
	require.NoError(t, err)
	assert.Equal(t, "All animals are equal.", content)
}

// TestBuildOptions_ThinkingAndSchema verifies BuildOptions reads the
// thinking level and response schema out of config.Parameters and wires
// them onto the returned GenerateOptions.
func TestBuildOptions_ThinkingAndSchema(t *testing.T) {
	provider := New()
	schema := `{"type":"object","properties":{"a":{"type":"integer"}}}`

	cfg := &config.Config{
		Parameters: config.Parameters{
			Thinking:       "high",
			ResponseSchema: schema,
		},
		Format: config.Format{},
	}

	opts := provider.BuildOptions(cfg)
	require.Len(t, opts, 1)
	ga, ok := opts[0].(*GenerateOptions)
	require.True(t, ok)

	assert.Equal(t, common.ThinkingHigh, ga.Thinking)
	require.NotNil(t, ga.ResponseFormat)
	assert.Equal(t, "json_schema", ga.ResponseFormat.Type)
	assert.Equal(t, "response", ga.ResponseFormat.Name)
	assert.JSONEq(t, schema, string(ga.ResponseFormat.Schema))
	require.NotNil(t, ga.ResponseFormat.Strict)
	assert.True(t, *ga.ResponseFormat.Strict)
}

// TestSupportsEffort covers the allowlist for the output_config.effort
// parameter. note that opus 4.5 supports effort even though it uses the
// manual (enabled+budget_tokens) thinking shape.
func TestSupportsEffort(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{name: "opus 4.5 supports effort", modelID: "claude-opus-4-5", want: true},
		{name: "opus 4.6 supports effort", modelID: "claude-opus-4-6", want: true},
		{name: "opus 4.7 supports effort", modelID: "claude-opus-4-7", want: true},
		{name: "sonnet 4.6 supports effort", modelID: "claude-sonnet-4-6", want: true},
		{name: "mythos preview supports effort", modelID: "claude-mythos-preview", want: true},
		{name: "sonnet 4.5 does not support effort", modelID: "claude-sonnet-4-5", want: false},
		{name: "haiku 4.5 does not support effort", modelID: "claude-haiku-4-5", want: false},
		{name: "3-7-sonnet does not support effort", modelID: "claude-3-7-sonnet-20241022", want: false},
		{name: "non-claude does not support effort", modelID: "gpt-5", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, supportsEffort(tt.modelID))
		})
	}
}

// TestBuildRequest_EffortOnOpus45 confirms that opus 4.5 gets both manual
// thinking (enabled+budget_tokens) and output_config.effort together —
// effort is independent of the adaptive-thinking routing.
func TestBuildRequest_EffortOnOpus45(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "user", Content: "Describe Snowball's role on the farm."},
	}

	t.Run("medium emits manual thinking plus medium effort", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingMedium))
		req, err := provider.BuildRequest(messages, "claude-opus-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)

		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "enabled", msgReq.Thinking.Type)
		assert.Equal(t, 4000, msgReq.Thinking.BudgetTokens)

		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "medium", msgReq.OutputConfig.Effort)
		// no schema requested, so format stays nil
		assert.Nil(t, msgReq.OutputConfig.Format)
	})

	t.Run("high on opus 4.5 falls back to high effort (max is 4.6+)", func(t *testing.T) {
		opts := NewGenerateOptions(WithThinking(common.ThinkingHigh))
		req, err := provider.BuildRequest(messages, "claude-opus-4-5", opts, slog.Default())
		require.NoError(t, err)
		msgReq, ok := req.(*MessagesRequest)
		require.True(t, ok)

		require.NotNil(t, msgReq.Thinking)
		assert.Equal(t, "enabled", msgReq.Thinking.Type)
		assert.Equal(t, 16000, msgReq.Thinking.BudgetTokens)

		require.NotNil(t, msgReq.OutputConfig)
		assert.Equal(t, "high", msgReq.OutputConfig.Effort)
	})
}

// TestBuildRequest_EffortAndSchemaCoexist confirms that a request with
// both a schema (WithSchema) and a thinking level populates both fields
// of output_config: format for the schema, effort for the thinking lever.
func TestBuildRequest_EffortAndSchemaCoexist(t *testing.T) {
	provider := New()
	messages := []common.Message{
		{Role: "user", Content: "Return the windmill quote."},
	}
	schema := []byte(`{"type":"object","properties":{"quote":{"type":"string"}}}`)

	opts := NewGenerateOptions(
		WithThinking(common.ThinkingHigh),
		WithSchema("animal_quote", schema),
	)
	req, err := provider.BuildRequest(messages, "claude-sonnet-4-6", opts, slog.Default())
	require.NoError(t, err)
	msgReq, ok := req.(*MessagesRequest)
	require.True(t, ok)

	require.NotNil(t, msgReq.Thinking)
	assert.Equal(t, "adaptive", msgReq.Thinking.Type)

	require.NotNil(t, msgReq.OutputConfig)
	// both output_config fields are populated together
	assert.Equal(t, "max", msgReq.OutputConfig.Effort)
	require.NotNil(t, msgReq.OutputConfig.Format)
	assert.Equal(t, "json_schema", msgReq.OutputConfig.Format.Type)
	assert.Equal(t, "animal_quote", msgReq.OutputConfig.Format.Name)
	assert.JSONEq(t, string(schema), string(msgReq.OutputConfig.Format.Schema))
}
