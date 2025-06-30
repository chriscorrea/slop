package common

import (
	"strings"
	"testing"
)

func TestValidateJSONResponse(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		config      *GenerateOptions
		expectError bool
		errorText   string
	}{
		{
			name:        "No JSON validation required - nil ResponseFormat",
			content:     `{"four_legs": "good"`, // validation should be skipped (but invalid JSON)
			config:      &GenerateOptions{},
			expectError: false,
		},
		{
			name:    "Valid JSON when required",
			content: `{"two_legs": "better"}`,
			config: &GenerateOptions{
				ResponseFormat: &ResponseFormat{Type: "json_object"},
			},
			expectError: false,
		},
		{
			name:    "Invalid JSON when required",
			content: `{"two_legs": "better"`, // missing closing brace
			config: &GenerateOptions{
				ResponseFormat: &ResponseFormat{Type: "json_object"},
			},
			expectError: true,
			errorText:   "API returned invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONResponse(tt.content, tt.config, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsJSONFormatRequested(t *testing.T) {
	tests := []struct {
		name     string
		config   *GenerateOptions
		expected bool
	}{
		{
			name: "JSON format requested",
			config: &GenerateOptions{
				ResponseFormat: &ResponseFormat{Type: "json_object"},
			},
			expected: true,
		},
		{
			name:     "ResponseFormat is nil",
			config:   &GenerateOptions{},
			expected: false,
		},
		{
			name: "Different format requested",
			config: &GenerateOptions{
				ResponseFormat: &ResponseFormat{Type: "text"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJSONFormatRequested(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
