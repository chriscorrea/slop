package template

import (
	"testing"
)

func TestProcessTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		userInput string
		expected  string
	}{
		{
			name:      "empty template returns user input",
			template:  "",
			userInput: "hello world",
			expected:  "hello world",
		},
		{
			name:      "empty template with empty input returns empty",
			template:  "",
			userInput: "",
			expected:  "",
		},
		{
			name:      "template with placeholder substitutes input",
			template:  "Please analyze: {input}",
			userInput: "this code",
			expected:  "Please analyze: this code",
		},
		{
			name:      "template with placeholder at start",
			template:  "{input} - analyze this",
			userInput: "Hello",
			expected:  "Hello - analyze this",
		},
		{
			name:      "template with multiple placeholders",
			template:  "Review {input} and provide feedback on {input}",
			userInput: "this code",
			expected:  "Review this code and provide feedback on this code",
		},
		{
			name:      "template without placeholder prepends",
			template:  "Code review:",
			userInput: "function main() {}",
			expected:  "Code review:\nfunction main() {}",
		},
		{
			name:      "template without placeholder and empty input returns template",
			template:  "Generate a summary",
			userInput: "",
			expected:  "Generate a summary",
		},
		{
			name:      "template with placeholder and empty input",
			template:  "Analyze this: {input}",
			userInput: "",
			expected:  "Analyze this: ",
		},
		{
			name:      "complex template with surrounding text",
			template:  "As a senior developer, please review {input} and focus on security issues.",
			userInput: "the attached Python script",
			expected:  "As a senior developer, please review the attached Python script and focus on security issues.",
		},
		{
			name:      "multi-line template with placeholder substitutes input in middle",
			template:  "Please analyze these windmill plans:\n{input}\nProvide detailed feedback.",
			userInput: "build it with care",
			expected:  "Please analyze these windmill plans:\nbuild it with care\nProvide detailed feedback.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessTemplate(tt.template, tt.userInput)
			if result != tt.expected {
				t.Errorf("ProcessTemplate(%q, %q) = %q; want %q", tt.template, tt.userInput, result, tt.expected)
			}
		})
	}
}

func TestHasPlaceholder(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected bool
	}{
		{
			name:     "template with placeholder",
			template: "Analyze {input} carefully",
			expected: true,
		},
		{
			name:     "template without placeholder",
			template: "Generate a summary",
			expected: false,
		},
		{
			name:     "empty template",
			template: "",
			expected: false,
		},
		{
			name:     "template with partial placeholder",
			template: "Look at {something} here",
			expected: false,
		},
		{
			name:     "template with multiple placeholders",
			template: "Compare {input} with {input}",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPlaceholder(tt.template)
			if result != tt.expected {
				t.Errorf("HasPlaceholder(%q) = %v; want %v", tt.template, result, tt.expected)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid template with placeholder",
			template: "Review {input}",
			wantErr:  false,
		},
		{
			name:     "valid template without placeholder",
			template: "Generate code",
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate(%q) error = %v; wantErr %v", tt.template, err, tt.wantErr)
			}
		})
	}
}