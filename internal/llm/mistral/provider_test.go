package mistral

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"

	"github.com/stretchr/testify/assert"
)

func TestCreateClient(t *testing.T) {
	provider := New()
	logger := slog.Default()

	tests := []struct {
		name          string
		config        *config.Config
		expectError   bool
		errorContains string
	}{
		{
			name: "Success with valid API key",
			config: &config.Config{
				Providers: config.Providers{
					Mistral: config.Mistral{
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
			name: "Success with custom base URL",
			config: &config.Config{
				Providers: config.Providers{
					Mistral: config.Mistral{
						BaseProvider: config.BaseProvider{
							APIKey:  "test-api-key",
							BaseUrl: "https://custom.mistral.api",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Failure with missing API key",
			config: &config.Config{
				Providers: config.Providers{
					Mistral: config.Mistral{
						BaseProvider: config.BaseProvider{
							APIKey: "",
						},
					},
				},
			},
			expectError:   true,
			errorContains: "Mistral API key is required",
		},
		{
			name:          "Failure with nil config",
			config:        nil,
			expectError:   true,
			errorContains: "config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := provider.CreateClient(tt.config, logger)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
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
		{Role: "user", Content: "Hello"},
	}
	modelName := "mistral-medium"

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
				assert.Equal(t, common.BoolPtr(false), chatReq.Stream)
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
				RandomSeed: common.IntPtr(42),
			},
			validate: func(t *testing.T, request interface{}) {
				chatReq, ok := request.(*ChatRequest)
				assert.True(t, ok, "Request should be *ChatRequest")
				assert.Equal(t, modelName, chatReq.Model)
				assert.Equal(t, messages, chatReq.Messages)

				// Verify parameter mapping
				assert.Equal(t, common.Float64Ptr(0.8), chatReq.Temperature)
				assert.Equal(t, common.IntPtr(1000), chatReq.MaxTokens)
				assert.Equal(t, common.Float64Ptr(0.9), chatReq.TopP)
				assert.Equal(t, []string{"STOP"}, chatReq.Stop)
				assert.Equal(t, common.IntPtr(42), chatReq.RandomSeed) // Mistral-specific field
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
				assert.NotNil(t, chatReq.ResponseFormat)
				assert.Equal(t, "json_object", chatReq.ResponseFormat.Type)
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
				"choices": [
					{
						"message": {
							"content": "Hello! How can I help you today?"
						}
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 15,
					"total_tokens": 25
				}
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
				"choices": [
					{
						"message": {
							"content": "Simple response"
						}
					}
				]
			}`),
			expectError:     false,
			expectedContent: "Simple response",
			expectedUsage: &common.Usage{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0,
			},
		},
		{
			name: "Response with empty choices",
			responseBody: []byte(`{
				"choices": []
			}`),
			expectError: true,
		},
		{
			name: "Malformed JSON response",
			responseBody: []byte(`{
				"choices": [
					{
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
			name:             "Unauthorized error",
			statusCode:       http.StatusUnauthorized,
			responseBody:     []byte(`{"error": {"message": "Invalid API key"}}`),
			expectedContains: []string{"Mistral API authentication failed", "API key"},
		},
		{
			name:             "Rate limit error",
			statusCode:       http.StatusTooManyRequests,
			responseBody:     []byte(`{"error": {"message": "Rate limit exceeded"}}`),
			expectedContains: []string{"Mistral API rate limit exceeded", "try again later"},
		},
		{
			name:             "Structured error response",
			statusCode:       http.StatusBadRequest,
			responseBody:     []byte(`{"error": {"message": "Invalid model specified"}}`),
			expectedContains: []string{"Mistral API error", "Invalid model specified"},
		},
		{
			name:             "Malformed error response",
			statusCode:       http.StatusInternalServerError,
			responseBody:     []byte(`invalid json`),
			expectedContains: []string{"Mistral API request failed with status 500"},
		},
		{
			name:             "Empty error message in structured response",
			statusCode:       http.StatusBadRequest,
			responseBody:     []byte(`{"error": {"message": ""}}`),
			expectedContains: []string{"unknown API error occurred", "status 400"},
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

func TestSupportsReasoningEffort(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{"mistral-small-2603 baseline", "mistral-small-2603", true},
		{"mistral-small newer date", "mistral-small-2610", true},
		{"mistral-small older than cutoff", "mistral-small-2509", false},
		{"mistral-small with suffix", "mistral-small-2603-preview", true},
		{"magistral-medium", "magistral-medium-2509", false},
		{"mistral-medium legacy", "mistral-medium-2312", false},
		{"mistral-tiny", "mistral-tiny", false},
		{"empty string", "", false},
		{"uppercase still matches", "Mistral-Small-2603", true},
		{"no date suffix", "mistral-small-latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsReasoningEffort(tt.modelID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildRequest_ReasoningEffort(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "four legs good, two legs bad"}}

	tests := []struct {
		name           string
		modelName      string
		effort         *string
		wantWired      bool
		wantEffortText string
	}{
		{
			name:           "mistral-small-2603 wires high effort",
			modelName:      "mistral-small-2603",
			effort:         strPtr("high"),
			wantWired:      true,
			wantEffortText: "high",
		},
		{
			name:      "magistral-medium-2509 drops effort",
			modelName: "magistral-medium-2509",
			effort:    strPtr("high"),
			wantWired: false,
		},
		{
			name:      "mistral-tiny drops effort",
			modelName: "mistral-tiny",
			effort:    strPtr("medium"),
			wantWired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &GenerateOptions{ReasoningEffort: tt.effort}
			req, err := provider.BuildRequest(messages, tt.modelName, opts, slog.Default())
			assert.NoError(t, err)

			chatReq, ok := req.(*ChatRequest)
			assert.True(t, ok, "Request should be *ChatRequest")

			if tt.wantWired {
				assert.NotNil(t, chatReq.ReasoningEffort)
				assert.Equal(t, tt.wantEffortText, *chatReq.ReasoningEffort)
			} else {
				assert.Nil(t, chatReq.ReasoningEffort, "ReasoningEffort must not be sent for unsupported models")
			}
		})
	}
}

func TestBuildRequest_SchemaEnvelope(t *testing.T) {
	provider := New()
	messages := []common.Message{{Role: "user", Content: "Boxer builds the windmill"}}
	schema := []byte(`{"type":"object","properties":{"character":{"type":"string"}},"required":["character"]}`)

	commonOpts := common.NewGenerateOptions(common.WithSchema("snowball_schema", schema))
	opts := &GenerateOptions{GenerateOptions: *commonOpts}

	req, err := provider.BuildRequest(messages, "mistral-small-2603", opts, slog.Default())
	assert.NoError(t, err)

	chatReq, ok := req.(*ChatRequest)
	assert.True(t, ok, "Request should be *ChatRequest")

	assert.NotNil(t, chatReq.ResponseFormat)
	assert.Equal(t, "json_schema", chatReq.ResponseFormat.Type)
	assert.NotNil(t, chatReq.ResponseFormat.JSONSchema, "json_schema envelope must be populated")
	assert.Equal(t, "snowball_schema", chatReq.ResponseFormat.JSONSchema.Name)
	assert.NotNil(t, chatReq.ResponseFormat.JSONSchema.Strict)
	assert.True(t, *chatReq.ResponseFormat.JSONSchema.Strict)
	assert.JSONEq(t, string(schema), string(chatReq.ResponseFormat.JSONSchema.Schema))

	// confirm the on-wire shape nests schema under json_schema, matching
	// Mistral's OpenAI-compatible envelope
	wire, err := json.Marshal(chatReq.ResponseFormat)
	assert.NoError(t, err)
	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(wire, &decoded))
	assert.Equal(t, "json_schema", decoded["type"])
	nested, ok := decoded["json_schema"].(map[string]interface{})
	assert.True(t, ok, "json_schema must be a nested object")
	assert.Equal(t, "snowball_schema", nested["name"])
	assert.Equal(t, true, nested["strict"])
	assert.NotNil(t, nested["schema"])
}

// TestExtractMagistralThinking_Placeholder exercises the current
// pass-through behavior. Replace this test once the live Magistral API
// response format has been verified and real extraction is implemented.
func TestExtractMagistralThinking_Placeholder(t *testing.T) {
	input := "Snowball drafts plans for the windmill before explaining the blueprint."
	thinking, cleaned := extractMagistralThinking(input)
	assert.Equal(t, "", thinking, "placeholder must return empty thinking until live verification")
	assert.Equal(t, input, cleaned, "placeholder must pass content through unchanged")
}

func strPtr(s string) *string { return &s }

func TestProviderInterface(t *testing.T) {
	provider := New()

	t.Run("RequiresAPIKey", func(t *testing.T) {
		assert.True(t, provider.RequiresAPIKey())
	})

	t.Run("ProviderName", func(t *testing.T) {
		assert.Equal(t, "mistral", provider.ProviderName())
	})

	t.Run("Implements common.Provider interface", func(t *testing.T) {
		assert.Implements(t, (*common.Provider)(nil), provider)
	})

	t.Run("HandleConnectionError", func(t *testing.T) {
		originalErr := assert.AnError
		result := provider.HandleConnectionError(originalErr)
		assert.Equal(t, originalErr, result)
	})

	t.Run("CustomizeRequest", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "https://api.mistral.ai/v1/chat/completions", nil)
		err := provider.CustomizeRequest(req)

		assert.NoError(t, err)
		// Mistral uses standard endpoint, so no URL modification expected
		assert.Equal(t, "/v1/chat/completions", req.URL.Path)
	})
}
