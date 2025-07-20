package template

import (
	"strings"
)

const (
	// InputPlaceholder is the standard placeholder for user input
	InputPlaceholder = "{input}"
)

// ProcessTemplate processes a message template with user input
//
// If template includes {input} placeholder, replace it with user input. Otherwise:
//   - empty template will return user input unchanged
//   - if no {input} placeholder, append user input to template with a newline
func ProcessTemplate(template, userInput string) string {
	// if no template is provided, use user input directly
	if template == "" {
		return userInput
	}
	
	// if template contains the input placeholder, substitute it
	if strings.Contains(template, InputPlaceholder) {
		return strings.ReplaceAll(template, InputPlaceholder, userInput)
	}
	
	// if no placeholder and no user input, return template only
	if userInput == "" {
		return template
	}
	
	// if no placeholder but has user input, prepend template
	return template + "\n" + userInput
}

// HasPlaceholder checks if a template contains the input placeholder
func HasPlaceholder(template string) bool {
	return strings.Contains(template, InputPlaceholder)
}

// ValidateTemplate validates a template string
func ValidateTemplate(template string) error {
	// templates are always valid in current implementation
	// this function exists for future extensibility
	return nil
}