package common

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
)

// HTTP utility functions provide standardized HTTP operations and error handling

// CreateJSONRequest creates a standard JSON API request with auth headers
func CreateJSONRequest(ctx context.Context, url, apiKey string, jsonData []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set standard headers for JSON API requests
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req, nil
}

// BuildChatCompletionsURL leverages how most providers use the same endpoint pattern
func BuildChatCompletionsURL(baseURL string) string {
	return fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(baseURL, "/"))
}
