package common

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogging_NilLogger(t *testing.T) {
	// sample data for testing
	sampleMessages := []Message{{Role: "user", Content: "I ate civilization. It poisoned me; I was defiled."}}
	sampleConfig := &GenerateOptions{
		Temperature: func() *float64 { v := 0.7; return &v }(),
		MaxTokens:   func() *int { v := 1932; return &v }(),
	}
	sampleUsage := Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30}
	sampleError := errors.New("test error")

	// all logging functions should be safe with nil logger
	assert.NotPanics(t, func() {
		LogAPIRequest(nil, "test-provider", "test-model", sampleMessages, sampleConfig)
	}, "LogAPIRequest with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogHTTPResponse(nil, 200, 123)
	}, "LogHTTPResponse with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogRawResponse(nil, "response body", 200)
	}, "LogRawResponse with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogTokenUsage(nil, "resp-123", sampleUsage)
	}, "LogTokenUsage with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogRequestCompletion(nil, 500)
	}, "LogRequestCompletion with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogRequestExecution(nil, "https://api.test.com", 3)
	}, "LogRequestExecution with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogRequestFailure(nil, sampleError, 3)
	}, "LogRequestFailure with nil logger should not panic")

	assert.NotPanics(t, func() {
		LogJSONUnmarshalError(nil, sampleError, "invalid json")
	}, "LogJSONUnmarshalError with nil logger should not panic")
}

func TestLogging_OutputContent(t *testing.T) {
	tests := []struct {
		name          string
		logFunc       func(*slog.Logger)
		expectedTexts []string
		expectedLevel string
	}{
		{
			name: "LogAPIRequest",
			logFunc: func(logger *slog.Logger) {
				messages := []Message{{Role: "user", Content: "Ending is better than mending."}}
				config := &GenerateOptions{
					Temperature: func() *float64 { v := 0.8; return &v }(),
					MaxTokens:   func() *int { v := 1024; return &v }(),
					TopP:        func() *float64 { v := 0.9; return &v }(),
				}
				LogAPIRequest(logger, "entropic", "sestina-2", messages, config)
			},
			expectedTexts: []string{"Sending request to entropic API", "sestina-2", "message_count=1", "temperature=0.8", "max_tokens=1024", "top_p=0.9"},
			expectedLevel: "DEBUG",
		},
		{
			name: "LogHTTPResponse",
			logFunc: func(logger *slog.Logger) {
				LogHTTPResponse(logger, 200, 1500)
			},
			expectedTexts: []string{"Received API response", "status_code=200", "body_length=1500"},
			expectedLevel: "DEBUG",
		},
		{
			name: "LogRawResponse",
			logFunc: func(logger *slog.Logger) {
				LogRawResponse(logger, `{"content":"Pain was a fascinating horror"}`, 200)
			},
			expectedTexts: []string{"Raw API response", "body", `content`, "status_code=200"},
			expectedLevel: "DEBUG",
		},
		{
			name: "LogTokenUsage",
			logFunc: func(logger *slog.Logger) {
				usage := Usage{PromptTokens: 15, CompletionTokens: 25, TotalTokens: 40}
				LogTokenUsage(logger, "resp-456", usage)
			},
			expectedTexts: []string{"Parsed API response", "response_id=resp-456", "prompt_tokens=15", "completion_tokens=25", "total_tokens=40"},
			expectedLevel: "DEBUG",
		},
		{
			name: "LogRequestCompletion",
			logFunc: func(logger *slog.Logger) {
				LogRequestCompletion(logger, 750)
			},
			expectedTexts: []string{"API request completed successfully", "response_length=750"},
			expectedLevel: "DEBUG",
		},
		{
			name: "LogRequestExecution",
			logFunc: func(logger *slog.Logger) {
				LogRequestExecution(logger, "https://api.entropic.com/v1/messages", 5)
			},
			expectedTexts: []string{"Executing API request", "url=https://api.entropic.com/v1/messages", "max_retries=5"},
			expectedLevel: "DEBUG",
		},
		{
			name: "LogRequestFailure",
			logFunc: func(logger *slog.Logger) {
				err := errors.New("connection timeout")
				LogRequestFailure(logger, err, 3)
			},
			expectedTexts: []string{"API request failed after retries", "error", "connection timeout", "max_retries=3"},
			expectedLevel: "ERROR",
		},
		{
			name: "LogJSONUnmarshalError",
			logFunc: func(logger *slog.Logger) {
				err := errors.New("invalid character '}' looking for beginning of value")
				LogJSONUnmarshalError(logger, err, `{"incomplete": }`)
			},
			expectedTexts: []string{"Failed to unmarshal JSON response", "error", "invalid character", "response_body", "incomplete"},
			expectedLevel: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup buffer to capture log output
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug, // Capture all levels
			}))

			// execute the logging function
			tt.logFunc(logger)

			// get the logged output
			output := buf.String()

			// assert expected level appears in output
			assert.Contains(t, output, tt.expectedLevel, "Expected log level %s in output", tt.expectedLevel)

			// assert all expected text content appears
			for _, expectedText := range tt.expectedTexts {
				assert.Contains(t, output, expectedText, "Expected text %q to appear in log output", expectedText)
			}
		})
	}
}

func TestLogging_OutputContent_EmptyConfig(t *testing.T) {
	// test LogAPIRequest with minimal config / no optional fields
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	messages := []Message{{Role: "user", Content: "test"}}
	emptyConfig := &GenerateOptions{} // no Temperature, MaxTokens, or TopP set

	LogAPIRequest(logger, "cohere", "command-r", messages, emptyConfig)

	output := buf.String()

	// should contain basic fields but not optional ones
	assert.Contains(t, output, "Sending request to cohere API")
	assert.Contains(t, output, "command-r")
	assert.Contains(t, output, "message_count=1")

	// should NOT contain optional fields
	assert.NotContains(t, output, "temperature")
	assert.NotContains(t, output, "max_tokens")
	assert.NotContains(t, output, "top_p")
}
