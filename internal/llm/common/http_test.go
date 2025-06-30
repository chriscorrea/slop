package common

import (
	"context"
	"io"
	"testing"
)

func TestCreateJSONRequest(t *testing.T) {
	ctx := context.Background()
	url := "https://api.example.com/endpoint"
	apiKey := "test-api-key"
	jsonData := []byte(`{"key":"value"}`)
	req, err := CreateJSONRequest(ctx, url, apiKey, jsonData)

	// checking that the HTTP method, URL, headers (like Content-Type and Authorization), and request body are all set as expected
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("expected method POST, got %s", req.Method)
	}
	if req.URL.String() != url {
		t.Errorf("expected URL %s, got %s", url, req.URL.String())
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Authorization") != "Bearer "+apiKey {
		t.Errorf("expected Authorization Bearer %s, got %s", apiKey, req.Header.Get("Authorization"))
	}

	// check if json data is set correctly
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	expectedBody := `{"key":"value"}`
	if string(body) != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, string(body))
	}
}

func TestBuildChatCompletionsURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "example base URL",
			baseURL:  "https://api.example.com/v1",
			expected: "https://api.example.com/v1/chat/completions",
		},
		{
			name:     "base URL with trailing slash",
			baseURL:  "https://api.example.com/",
			expected: "https://api.example.com/chat/completions",
		},
		{
			name:     "localhost with port URL",
			baseURL:  "http://localhost:8080",
			expected: "http://localhost:8080/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildChatCompletionsURL(tt.baseURL)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
