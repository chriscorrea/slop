package openai

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
	assert.Equal(t, "openai", provider.ProviderName())
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
					OpenAI: config.OpenAI{},
				},
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Providers: config.Providers{
					OpenAI: config.OpenAI{
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
					OpenAI: config.OpenAI{
						BaseProvider: config.BaseProvider{
							APIKey:  "test-api-key",
							BaseUrl: "https://custom.openai.com/v1",
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
					Temperature:   0.7,
					MaxTokens:     1000,
					TopP:          0.9,
					StopSequences: []string{"end", "stop"},
					Seed:          common.IntPtr(42),
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
		{Role: "user", Content: "Hello, how are you?"},
	}
	modelName := "gpt-4"

	tests := []struct {
		name     string
		options  interface{}
		expected *ChatRequest
	}{
		{
			name:    "nil options",
			options: nil,
			expected: &ChatRequest{
				Model:    modelName,
				Messages: messages,
				Stream:   common.BoolPtr(false),
			},
		},
		{
			name: "with generate options",
			options: NewGenerateOptions(
				WithTemperature(0.7),
				WithMaxTokens(1000),
				WithFrequencyPenalty(0.5),
				WithSeed(42),
				WithTools([]Tool{
					{
						Type: "function",
						Function: Function{
							Name:        "test_function",
							Description: "A test function",
						},
					},
				}),
			),
			expected: &ChatRequest{
				Model:               modelName,
				Messages:            messages,
				Temperature:         common.Float64Ptr(0.7),
				MaxCompletionTokens: common.IntPtr(1000),
				FrequencyPenalty:    common.Float64Ptr(0.5),
				Seed:                common.IntPtr(42),
				Tools: []Tool{
					{
						Type: "function",
						Function: Function{
							Name:        "test_function",
							Description: "A test function",
						},
					},
				},
				Stream: common.BoolPtr(false),
			},
		},
		{
			name:    "invalid options type",
			options: "invalid",
			expected: &ChatRequest{
				Model:    modelName,
				Messages: messages,
				Stream:   common.BoolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := provider.BuildRequest(messages, modelName, tt.options, slog.Default())

			assert.NoError(t, err)
			assert.NotNil(t, request)

			chatReq, ok := request.(*ChatRequest)
			require.True(t, ok)

			assert.Equal(t, tt.expected.Model, chatReq.Model)
			assert.Equal(t, tt.expected.Messages, chatReq.Messages)
			assert.Equal(t, tt.expected.Stream, chatReq.Stream)

			if tt.expected.Temperature != nil {
				require.NotNil(t, chatReq.Temperature)
				assert.Equal(t, *tt.expected.Temperature, *chatReq.Temperature)
			}
			if tt.expected.MaxCompletionTokens != nil {
				require.NotNil(t, chatReq.MaxCompletionTokens)
				assert.Equal(t, *tt.expected.MaxCompletionTokens, *chatReq.MaxCompletionTokens)
			}
			if tt.expected.FrequencyPenalty != nil {
				require.NotNil(t, chatReq.FrequencyPenalty)
				assert.Equal(t, *tt.expected.FrequencyPenalty, *chatReq.FrequencyPenalty)
			}
			if tt.expected.Seed != nil {
				require.NotNil(t, chatReq.Seed)
				assert.Equal(t, *tt.expected.Seed, *chatReq.Seed)
			}
			if len(tt.expected.Tools) > 0 {
				assert.Equal(t, tt.expected.Tools, chatReq.Tools)
			}
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
				"choices": [
					{
						"message": {
							"content": "Hello! I'm doing well, thank you for asking."
						}
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 15,
					"total_tokens": 25
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
			name:          "invalid JSON",
			responseBody:  `{invalid json}`,
			expectedError: true,
		},
		{
			name: "no choices",
			responseBody: `{
				"choices": [],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
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
			expectedSubstr: "OpenAI API authentication failed",
		},
		{
			name:           "rate limit exceeded",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   "",
			expectedSubstr: "OpenAI API rate limit exceeded",
		},
		{
			name:           "structured error response",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": {"message": "Invalid request format"}}`,
			expectedSubstr: "OpenAI API error: Invalid request format",
		},
		{
			name:           "malformed error response",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "Internal server error",
			expectedSubstr: "OpenAI API request failed with status 500",
		},
		{
			name:           "empty error message",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": {"message": ""}}`,
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
	req := httptest.NewRequest("POST", "https://api.openai.com/v1/chat/completions", nil)

	err := provider.CustomizeRequest(req)
	assert.NoError(t, err)
}

// Integration test with mock HTTP server
func TestProvider_Integration(t *testing.T) {
	// create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// parse request body
		var reqBody ChatRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, "gpt-4", reqBody.Model)
		assert.Len(t, reqBody.Messages, 1)
		assert.Equal(t, "user", reqBody.Messages[0].Role)
		assert.Equal(t, "test message", reqBody.Messages[0].Content)

		// send response
		response := common.ChatResponse{
			Choices: []common.Choice{
				{
					Message: common.Message{
						Content: "test response",
					},
				},
			},
			Usage: common.Usage{
				PromptTokens:     5,
				CompletionTokens: 2,
				TotalTokens:      7,
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
			OpenAI: config.OpenAI{
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

	content, err := client.Generate(context.Background(), messages, "gpt-4")
	require.NoError(t, err)
	assert.Equal(t, "test response", content)
}
