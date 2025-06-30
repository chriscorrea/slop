package common

import (
	"fmt"
	"log/slog"
)

// LogAPIRequest logs standardized API request information
// for consistent request logging across all providers
func LogAPIRequest(logger *slog.Logger, providerName, modelName string, messages []Message, config *GenerateOptions) {
	if logger == nil {
		return
	}

	// build the log message with common fields
	args := []interface{}{
		"model", modelName,
		"message_count", len(messages),
	}

	// add some optional parameters if they exist
	if config.Temperature != nil {
		args = append(args, "temperature", *config.Temperature)
	}
	if config.MaxTokens != nil {
		args = append(args, "max_tokens", *config.MaxTokens)
	}
	if config.TopP != nil {
		args = append(args, "top_p", *config.TopP)
	}

	logger.Debug(fmt.Sprintf("Sending request to %s API", providerName), args...)
}

// LogHTTPResponse logs basic HTTP response information
func LogHTTPResponse(logger *slog.Logger, statusCode int, bodyLength int) {
	if logger == nil {
		return
	}
	logger.Debug("Received API response",
		"status_code", statusCode,
		"body_length", bodyLength)
}

// LogRawResponse logs the raw API response body for debugging
func LogRawResponse(logger *slog.Logger, body string, statusCode int) {
	if logger == nil {
		return
	}
	logger.Debug("Raw API response",
		"body", body,
		"status_code", statusCode)
}

// LogTokenUsage logs token consumption from a standard Usage struct
func LogTokenUsage(logger *slog.Logger, responseID string, usage Usage) {
	if logger == nil {
		return
	}
	logger.Debug("Parsed API response",
		"response_id", responseID,
		"prompt_tokens", usage.PromptTokens,
		"completion_tokens", usage.CompletionTokens,
		"total_tokens", usage.TotalTokens)
}

// LogRequestCompletion logs successful request completion
func LogRequestCompletion(logger *slog.Logger, contentLength int) {
	if logger == nil {
		return
	}
	logger.Debug("API request completed successfully",
		"response_length", contentLength)
}

// LogRequestExecution logs request execution details
func LogRequestExecution(logger *slog.Logger, url string, maxRetries int) {
	if logger == nil {
		return
	}
	logger.Debug("Executing API request",
		"url", url,
		"max_retries", maxRetries)
}

// LogRequestFailure logs request execution failures
func LogRequestFailure(logger *slog.Logger, err error, maxRetries int) {
	if logger == nil {
		return
	}
	logger.Error("API request failed after retries",
		"error", err,
		"max_retries", maxRetries)
}

// LogJSONUnmarshalError logs JSON parsing errors with context
func LogJSONUnmarshalError(logger *slog.Logger, err error, responseBody string) {
	if logger == nil {
		return
	}
	logger.Error("Failed to unmarshal JSON response",
		"error", err,
		"response_body", responseBody)
}
