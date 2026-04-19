package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
	assert.Equal(t, "groq", provider.ProviderName())
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
					Groq: config.Groq{},
				},
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Providers: config.Providers{
					Groq: config.Groq{
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
					Groq: config.Groq{
						BaseProvider: config.BaseProvider{
							APIKey:  "test-api-key",
							BaseUrl: "https://custom.groq.com/v1",
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
	modelName := "llama-3.1-8b-instant"

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
			),
			expected: &ChatRequest{
				Model:            modelName,
				Messages:         messages,
				Temperature:      common.Float64Ptr(0.7),
				MaxTokens:        common.IntPtr(1000),
				FrequencyPenalty: common.Float64Ptr(0.5),
				Seed:             common.IntPtr(42),
				Stream:           common.BoolPtr(false),
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
			if tt.expected.MaxTokens != nil {
				require.NotNil(t, chatReq.MaxTokens)
				assert.Equal(t, *tt.expected.MaxTokens, *chatReq.MaxTokens)
			}
			if tt.expected.FrequencyPenalty != nil {
				require.NotNil(t, chatReq.FrequencyPenalty)
				assert.Equal(t, *tt.expected.FrequencyPenalty, *chatReq.FrequencyPenalty)
			}
			if tt.expected.Seed != nil {
				require.NotNil(t, chatReq.Seed)
				assert.Equal(t, *tt.expected.Seed, *chatReq.Seed)
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
			expectedSubstr: "Groq API authentication failed",
		},
		{
			name:           "rate limit exceeded",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   "",
			expectedSubstr: "Groq API rate limit exceeded",
		},
		{
			name:           "structured error response",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": {"message": "Invalid request format"}}`,
			expectedSubstr: "Groq API error: Invalid request format",
		},
		{
			name:           "malformed error response",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "Internal server error",
			expectedSubstr: "Groq API request failed with status 500",
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
	req := httptest.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", nil)

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

		assert.Equal(t, "llama-3.1-8b-instant", reqBody.Model)
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
			Groq: config.Groq{
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

	content, err := client.Generate(context.Background(), messages, "llama-3.1-8b-instant")
	require.NoError(t, err)
	assert.Equal(t, "test response", content)
}

// TestSupportsReasoning covers the model-ID-aware gate for reasoning_format
func TestSupportsReasoning(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{"empty", "", false},
		{"qwen-3-32b", "qwen-3-32b", true},
		{"qwen-3-prefix uppercase", "Qwen-3-32B", true},
		{"qwen3 dashed variant", "qwen3-32b", true},
		{"deepseek r1 distill llama", "deepseek-r1-distill-llama-70b", true},
		{"deepseek r1 distill qwen", "deepseek-r1-distill-qwen-32b", true},
		{"compound", "groq/compound", false},
		{"compound trimmed casing", " Groq/Compound ", false},
		{"llama 3.3 70b", "llama-3.3-70b-versatile", false},
		{"llama 3.1 8b instant", "llama-3.1-8b-instant", false},
		{"mixtral", "mixtral-8x7b-32768", false},
		{"unrelated id", "some-other-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsReasoning(tt.modelID)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestTranslateThinkingLevel covers the ThinkingLevel to reasoning_format map
func TestTranslateThinkingLevel(t *testing.T) {
	assert.Equal(t, "", translateThinkingLevel(common.ThinkingOff))
	assert.Equal(t, "parsed", translateThinkingLevel(common.ThinkingMedium))
	assert.Equal(t, "parsed", translateThinkingLevel(common.ThinkingHigh))
}

// TestProvider_BuildRequest_ReasoningFormat confirms that ReasoningFormat is
// wired through only for reasoning-capable model IDs. Plain models and the
// agentic Compound model must not emit the parameter
func TestProvider_BuildRequest_ReasoningFormat(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "Snowball gallops toward the windmill"}}

	tests := []struct {
		name      string
		modelName string
		wantSet   bool
	}{
		{"qwen-3 reasoning model", "qwen-3-32b", true},
		{"deepseek r1 distill reasoning model", "deepseek-r1-distill-llama-70b", true},
		{"compound skips the field", "groq/compound", false},
		{"plain llama skips the field", "llama-3.3-70b-versatile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewGenerateOptions(WithReasoningFormat("parsed"))

			request, err := provider.BuildRequest(messages, tt.modelName, opts, slog.Default())
			require.NoError(t, err)

			chatReq, ok := request.(*ChatRequest)
			require.True(t, ok)

			if tt.wantSet {
				require.NotNil(t, chatReq.ReasoningFormat)
				assert.Equal(t, "parsed", *chatReq.ReasoningFormat)
			} else {
				assert.Nil(t, chatReq.ReasoningFormat)
			}
		})
	}
}

// TestProvider_BuildOptions_ThinkingAndSchema verifies that the cross-provider
// thinking level and the pre-resolved response schema are both translated
// into the Groq-specific GenerateOptions during BuildOptions
func TestProvider_BuildOptions_ThinkingAndSchema(t *testing.T) {
	provider := New()
	schema := `{"type":"object","properties":{"quote":{"type":"string"}}}`

	cfg := &config.Config{
		Parameters: config.Parameters{
			Thinking:       "high",
			ResponseSchema: schema,
		},
	}

	options := provider.BuildOptions(cfg)
	require.Len(t, options, 1)

	groqOpts, ok := options[0].(*GenerateOptions)
	require.True(t, ok)

	require.NotNil(t, groqOpts.ReasoningFormat)
	assert.Equal(t, "parsed", *groqOpts.ReasoningFormat)

	require.NotNil(t, groqOpts.ResponseFormat)
	assert.Equal(t, "json_schema", groqOpts.ResponseFormat.Type)
	assert.Equal(t, "response", groqOpts.ResponseFormat.Name)
	require.NotNil(t, groqOpts.ResponseFormat.Strict)
	assert.True(t, *groqOpts.ResponseFormat.Strict)
	assert.JSONEq(t, schema, string(groqOpts.ResponseFormat.Schema))
}

// TestProvider_BuildRequest_JSONSchemaEnvelope verifies that a schema set via
// WithSchema serializes as the OpenAI-compatible json_schema envelope on the
// wire, with the nested name/schema/strict fields Groq expects
func TestProvider_BuildRequest_JSONSchemaEnvelope(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "describe the cowshed"}}
	schema := []byte(`{"type":"object","properties":{"barn":{"type":"string"}}}`)

	base := common.NewGenerateOptions(common.WithSchema("barn_schema", schema))
	opts := &GenerateOptions{GenerateOptions: *base}

	request, err := provider.BuildRequest(messages, "llama-3.3-70b-versatile", opts, slog.Default())
	require.NoError(t, err)

	chatReq, ok := request.(*ChatRequest)
	require.True(t, ok)

	require.NotNil(t, chatReq.ResponseFormat)
	assert.Equal(t, "json_schema", chatReq.ResponseFormat.Type)
	require.NotNil(t, chatReq.ResponseFormat.JSONSchema)
	assert.Equal(t, "barn_schema", chatReq.ResponseFormat.JSONSchema.Name)
	require.NotNil(t, chatReq.ResponseFormat.JSONSchema.Strict)
	assert.True(t, *chatReq.ResponseFormat.JSONSchema.Strict)
	assert.JSONEq(t, string(schema), string(chatReq.ResponseFormat.JSONSchema.Schema))

	// also verify the serialized JSON shape matches the expected wire envelope
	wire, err := json.Marshal(chatReq.ResponseFormat)
	require.NoError(t, err)

	var decoded struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Name   string          `json:"name"`
			Schema json.RawMessage `json:"schema"`
			Strict *bool           `json:"strict"`
		} `json:"json_schema"`
	}
	require.NoError(t, json.Unmarshal(wire, &decoded))

	assert.Equal(t, "json_schema", decoded.Type)
	assert.Equal(t, "barn_schema", decoded.JSONSchema.Name)
	require.NotNil(t, decoded.JSONSchema.Strict)
	assert.True(t, *decoded.JSONSchema.Strict)
	assert.JSONEq(t, string(schema), string(decoded.JSONSchema.Schema))
}

// TestProvider_BuildRequest_JSONObjectPassthrough verifies the simpler
// {"type":"json_object"} wire form survives unchanged
func TestProvider_BuildRequest_JSONObjectPassthrough(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "return json"}}

	opts := NewGenerateOptions(WithJSONFormat())

	request, err := provider.BuildRequest(messages, "llama-3.3-70b-versatile", opts, slog.Default())
	require.NoError(t, err)

	chatReq, ok := request.(*ChatRequest)
	require.True(t, ok)

	require.NotNil(t, chatReq.ResponseFormat)
	assert.Equal(t, "json_object", chatReq.ResponseFormat.Type)
	assert.Nil(t, chatReq.ResponseFormat.JSONSchema)
}

// TestProvider_BuildRequest_CompoundEmitsDebugLog confirms that selecting the
// agentic Compound model surfaces a debug-level log explaining that latency
// will vary. Non-compound models must stay silent on this line
func TestProvider_BuildRequest_CompoundEmitsDebugLog(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "four legs good"}}

	tests := []struct {
		name       string
		modelName  string
		wantLogged bool
	}{
		{"compound logs", "groq/compound", true},
		{"mixed case compound logs", "Groq/Compound", true},
		{"plain llama stays silent", "llama-3.3-70b-versatile", false},
		{"qwen reasoning stays silent", "qwen-3-32b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			logger := slog.New(handler)

			_, err := provider.BuildRequest(messages, tt.modelName, nil, logger)
			require.NoError(t, err)

			logged := strings.Contains(buf.String(), "groq/compound may perform web search or code execution")
			assert.Equal(t, tt.wantLogged, logged, "log output: %s", buf.String())
		})
	}
}
