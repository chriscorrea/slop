package cohere

import (
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
					Cohere: config.Cohere{
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
					Cohere: config.Cohere{
						BaseProvider: config.BaseProvider{
							APIKey:  "test-api-key",
							BaseUrl: "https://custom.cohere.api",
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
					Cohere: config.Cohere{
						BaseProvider: config.BaseProvider{
							APIKey: "",
						},
					},
				},
			},
			expectError:   true,
			errorContains: "Cohere API key is required",
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
	modelName := "command-r"

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
				TopK:       common.IntPtr(50),
				Seed:       common.IntPtr(42),
				SafetyMode: common.StringPtr("strict"),
			},
			validate: func(t *testing.T, request interface{}) {
				chatReq, ok := request.(*ChatRequest)
				assert.True(t, ok, "Request should be *ChatRequest")
				assert.Equal(t, modelName, chatReq.Model)
				assert.Equal(t, messages, chatReq.Messages)

				// Verify parameter mapping
				assert.Equal(t, common.Float64Ptr(0.8), chatReq.Temperature)
				assert.Equal(t, common.IntPtr(1000), chatReq.MaxTokens)
				assert.Equal(t, common.Float64Ptr(0.9), chatReq.P) // TopP -> P mapping
				assert.Equal(t, []string{"STOP"}, chatReq.StopSequences)
				assert.Equal(t, common.IntPtr(50), chatReq.K) // TopK -> K mapping
				assert.Equal(t, common.IntPtr(42), chatReq.Seed)
				assert.Equal(t, common.StringPtr("strict"), chatReq.SafetyMode)
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
				"message": {
					"content": "Hello! How can I help you today?"
				},
				"usage": {
					"tokens": {
						"input_tokens": 10,
						"output_tokens": 15
					}
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
				"message": {
					"content": "Simple response"
				},
				"usage": {
					"tokens": {
						"input_tokens": 0,
						"output_tokens": 0
					}
				}
			}`),
			expectError:     false,
			expectedContent: "Simple response",
			expectedUsage:   nil,
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
			name:             "Unauthorized error",
			statusCode:       http.StatusUnauthorized,
			responseBody:     []byte(`{"message": "Invalid API key"}`),
			expectedContains: []string{"Cohere API authentication failed", "API key"},
		},
		{
			name:             "Rate limit error",
			statusCode:       http.StatusTooManyRequests,
			responseBody:     []byte(`{"message": "Rate limit exceeded"}`),
			expectedContains: []string{"Cohere API rate limit exceeded", "try again later"},
		},
		{
			name:             "Structured error response",
			statusCode:       http.StatusBadRequest,
			responseBody:     []byte(`{"message": "Invalid model specified"}`),
			expectedContains: []string{"Cohere API error", "Invalid model specified"},
		},
		{
			name:             "Malformed error response",
			statusCode:       http.StatusInternalServerError,
			responseBody:     []byte(`invalid json`),
			expectedContains: []string{"Cohere API request failed with status 500"},
		},
		{
			name:             "Empty error message in structured response",
			statusCode:       http.StatusBadRequest,
			responseBody:     []byte(`{"message": ""}`),
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

func TestBuildOptions(t *testing.T) {
	provider := New()

	tests := []struct {
		name     string
		config   *config.Config
		validate func(t *testing.T, options []interface{})
	}{
		{
			name: "Default config with zero values",
			config: &config.Config{
				Parameters: config.Parameters{
					Temperature:   0,   // zero value - should not create option
					MaxTokens:     0,   // zero value - should not create option
					TopP:          0,   // zero value - should not create option
					StopSequences: nil, // nil - should not create option
					Seed:          nil, // nil - should not create option
				},
				Format: config.Format{
					JSON: false, // false - should not create option
				},
			},
			validate: func(t *testing.T, options []interface{}) {
				assert.Len(t, options, 1, "Should return exactly one GenerateOptions object")

				genOpts, ok := options[0].(*GenerateOptions)
				assert.True(t, ok, "Should return *GenerateOptions")

				// All fields should be nil/zero since no options were set
				assert.Nil(t, genOpts.Temperature, "Temperature should be nil for zero value")
				assert.Nil(t, genOpts.MaxTokens, "MaxTokens should be nil for zero value")
				assert.Nil(t, genOpts.TopP, "TopP should be nil for zero value")
				assert.Nil(t, genOpts.Stop, "Stop should be nil for empty slice")
				assert.Nil(t, genOpts.Seed, "Seed should be nil")
				assert.Nil(t, genOpts.ResponseFormat, "ResponseFormat should be nil for JSON=false")
			},
		},
		{
			name: "Config with all parameters set",
			config: &config.Config{
				Parameters: config.Parameters{
					Temperature:   0.8,
					MaxTokens:     1932,
					TopP:          0.9,
					StopSequences: []string{"STOP", "END"},
					Seed:          func() *int { v := 42; return &v }(),
				},
				Format: config.Format{
					JSON: false,
				},
			},
			validate: func(t *testing.T, options []interface{}) {
				assert.Len(t, options, 1, "Should return exactly one GenerateOptions object")

				genOpts, ok := options[0].(*GenerateOptions)
				assert.True(t, ok, "Should return *GenerateOptions")

				// Verify all parameters are correctly translated
				assert.NotNil(t, genOpts.Temperature, "Temperature should be set")
				assert.Equal(t, 0.8, *genOpts.Temperature, "Temperature should be 0.8")

				assert.NotNil(t, genOpts.MaxTokens, "MaxTokens should be set")
				assert.Equal(t, 1932, *genOpts.MaxTokens, "MaxTokens should be 1932")

				assert.NotNil(t, genOpts.TopP, "TopP should be set")
				assert.Equal(t, 0.9, *genOpts.TopP, "TopP should be 0.9")

				assert.NotNil(t, genOpts.Stop, "Stop should be set")
				assert.Equal(t, []string{"STOP", "END"}, genOpts.Stop, "Stop sequences should match")

				assert.NotNil(t, genOpts.Seed, "Seed should be set")
				assert.Equal(t, 42, *genOpts.Seed, "Seed should be 42")

				assert.Nil(t, genOpts.ResponseFormat, "ResponseFormat should be nil for JSON=false")
			},
		},
		{
			name: "Config with JSON format enabled",
			config: &config.Config{
				Parameters: config.Parameters{
					Temperature: 0.7, // Include one other parameter to verify both work
				},
				Format: config.Format{
					JSON: true,
				},
			},
			validate: func(t *testing.T, options []interface{}) {
				assert.Len(t, options, 1, "Should return exactly one GenerateOptions object")

				genOpts, ok := options[0].(*GenerateOptions)
				assert.True(t, ok, "Should return *GenerateOptions")

				// verify JSON format is set
				assert.NotNil(t, genOpts.ResponseFormat, "ResponseFormat should be set for JSON=true")
				assert.Equal(t, "json_object", genOpts.ResponseFormat.Type, "ResponseFormat type should be json_object")

				// verify temperature is also set
				assert.NotNil(t, genOpts.Temperature, "Temperature should be set")
				assert.Equal(t, 0.7, *genOpts.Temperature, "Temperature should be 0.7")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := provider.BuildOptions(tt.config)
			tt.validate(t, options)
		})
	}
}
