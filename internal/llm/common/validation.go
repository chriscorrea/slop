package common

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// JSON validation functions for structured output responses
// these provide consistent validation behavior across all providers

// ValidateJSONResponse validates that content is valid JSON
// this should be called after receiving a response that was requested with JSON format
func ValidateJSONResponse(content string, config *GenerateOptions, logger *slog.Logger) error {
	// check if JSON validation is needed using the existing helper
	if !IsJSONFormatRequested(config) {
		return nil
	}

	if logger != nil {
		logger.Debug("Validating JSON response", "content", content)
	}

	var jsonTest interface{}
	if err := json.Unmarshal([]byte(content), &jsonTest); err != nil {
		if logger != nil {
			logger.Error("Invalid JSON in response", "error", err, "content", content)
		}
		return fmt.Errorf("API returned invalid JSON: %w. Response: %s", err, content)
	}

	if logger != nil {
		logger.Debug("JSON response validation passed")
	}

	return nil
}

// IsJSONFormatRequested checks if JSON structured output was requested in the configuration
func IsJSONFormatRequested(config *GenerateOptions) bool {
	return config.ResponseFormat != nil && config.ResponseFormat.Type == "json_object"
}

// ValidateJSONSchema is basic check that the given bytes are valid JSON
func ValidateJSONSchema(schema []byte) error {
	if len(schema) == 0 {
		return nil
	}
	var parsed interface{}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}
	return nil
}
