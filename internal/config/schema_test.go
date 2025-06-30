package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveKey(t *testing.T) {
	schema := DefaultConfigSchema()

	tests := []struct {
		name             string
		key              string
		expectedPath     string
		expectError      bool
		errorContainsAny []string
	}{
		{
			name:         "Valid temp alias resolves to canonical path",
			key:          "temp",
			expectedPath: "parameters.temperature",
			expectError:  false,
		},
		{
			name:         "Valid canonical path resolves to itself",
			key:          "parameters.temperature",
			expectedPath: "parameters.temperature",
			expectError:  false,
		},
		{
			name:         "Valid max-tokens alias",
			key:          "max-tokens",
			expectedPath: "parameters.max_tokens",
			expectError:  false,
		},
		{
			name:         "Valid canonical provider key",
			key:          "providers.anthropic.api_key",
			expectedPath: "providers.anthropic.api_key",
			expectError:  false,
		},
		{
			name:             "Invalid key returns error",
			key:              "nonexistent.key",
			expectedPath:     "",
			expectError:      true,
			errorContainsAny: []string{"invalid config key", "nonexistent.key"},
		},
		{
			name:             "Case sensitivity test - uppercase alias",
			key:              "Temp",
			expectedPath:     "",
			expectError:      true,
			errorContainsAny: []string{"invalid config key", "Temp"},
		},
		{
			name:             "Case sensitivity test (mixed case)",
			key:              "TEMPerature",
			expectedPath:     "",
			expectError:      true,
			errorContainsAny: []string{"invalid config key", "TEMPerature"},
		},
		{
			name:             "Empty key",
			key:              "",
			expectedPath:     "",
			expectError:      true,
			errorContainsAny: []string{"invalid config key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvedPath, err := schema.ResolveKey(tt.key)

			if tt.expectError {
				assert.Error(t, err)
				if len(tt.errorContainsAny) > 0 {
					errorStr := err.Error()
					for _, expectedSubstring := range tt.errorContainsAny {
						assert.Contains(t, errorStr, expectedSubstring)
					}
				}
				assert.Equal(t, tt.expectedPath, resolvedPath)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, resolvedPath)
			}
		})
	}
}

func TestValidateValue(t *testing.T) {
	schema := DefaultConfigSchema()

	tests := []struct {
		name             string
		path             string
		value            interface{}
		expectError      bool
		errorContainsAny []string
	}{
		// parameters.temperature (float64) tests
		{
			name:        "Valid temperature value",
			path:        "parameters.temperature",
			value:       0.577,
			expectError: false,
		},
		{
			name:             "Temperature invalid type - string",
			path:             "parameters.temperature",
			value:            "0.577",
			expectError:      true,
			errorContainsAny: []string{"expected float64", "got string"},
		},
		{
			name:             "Temperature out of range - too high",
			path:             "parameters.temperature",
			value:            3.14,
			expectError:      true,
			errorContainsAny: []string{"value must be between", "0.00", "1.00"},
		},
		{
			name:        "Temperature edge value - minimum",
			path:        "parameters.temperature",
			value:       0.0,
			expectError: false,
		},
		{
			name:        "Temperature edge value - maximum",
			path:        "parameters.temperature",
			value:       1.0,
			expectError: false,
		},
		{
			name:        "Valid max_tokens value",
			path:        "parameters.max_tokens",
			value:       4096,
			expectError: false,
		},
		{
			name:             "Max_tokens invalid type - float64",
			path:             "parameters.max_tokens",
			value:            4096.0,
			expectError:      true,
			errorContainsAny: []string{"expected int", "got float64"},
		},
		{
			name:             "Max_tokens out of range - too low",
			path:             "parameters.max_tokens",
			value:            0,
			expectError:      true,
			errorContainsAny: []string{"value must be between", "1", "100000"},
		},
		{
			name:        "Max_tokens edge value - minimum",
			path:        "parameters.max_tokens",
			value:       1,
			expectError: false,
		},
		{
			name:        "Max_tokens edge value - high",
			path:        "parameters.max_tokens",
			value:       10000,
			expectError: false,
		},
		{
			name:        "Valid seed value - pointer to int",
			path:        "parameters.seed",
			value:       42,
			expectError: false,
		},
		{
			name:        "Valid API key - non-empty string",
			path:        "providers.anthropic.api_key",
			value:       "sk-test-key-123",
			expectError: false,
		},
		{
			name:        "Valid API key - empty string",
			path:        "providers.anthropic.api_key",
			value:       "",
			expectError: false,
		},
		{
			name:             "Unknown config path",
			path:             "unknown.path",
			value:            "any value",
			expectError:      true,
			errorContainsAny: []string{"unknown config path", "unknown.path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateValue(tt.path, tt.value)

			if tt.expectError {
				assert.Error(t, err)
				if len(tt.errorContainsAny) > 0 {
					errorStr := err.Error()
					for _, expectedSubstring := range tt.errorContainsAny {
						assert.Contains(t, errorStr, expectedSubstring)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetFieldInfo(t *testing.T) {
	schema := DefaultConfigSchema()

	tests := []struct {
		name               string
		path               string
		expectError        bool
		expectedType       reflect.Type
		expectedDefault    interface{}
		expectedValidation bool // should validation function be present
		errorContainsAny   []string
	}{
		{
			name:               "Valid key - parameters.max_tokens",
			path:               "parameters.max_tokens",
			expectError:        false,
			expectedType:       reflect.TypeOf(int(0)),
			expectedDefault:    2048,
			expectedValidation: true,
		},
		{
			name:               "Valid key - parameters.temperature",
			path:               "parameters.temperature",
			expectError:        false,
			expectedType:       reflect.TypeOf(float64(0)),
			expectedDefault:    0.7,
			expectedValidation: true,
		},
		{
			name:               "Valid key - providers.anthropic.api_key",
			path:               "providers.anthropic.api_key",
			expectError:        false,
			expectedType:       reflect.TypeOf(""),
			expectedDefault:    "",
			expectedValidation: false, // no validation function for strings
		},
		{
			name:               "Valid key - parameters.seed",
			path:               "parameters.seed",
			expectError:        false,
			expectedType:       reflect.TypeOf((*int)(nil)).Elem(),
			expectedDefault:    nil,
			expectedValidation: true,
		},
		{
			name:             "Invalid key",
			path:             "nonexistent.path",
			expectError:      true,
			errorContainsAny: []string{"unknown config path", "nonexistent.path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldInfo, err := schema.GetFieldInfo(tt.path)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ConfigFieldInfo{}, fieldInfo)
				if len(tt.errorContainsAny) > 0 {
					errorStr := err.Error()
					for _, expectedSubstring := range tt.errorContainsAny {
						assert.Contains(t, errorStr, expectedSubstring)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, fieldInfo.Type)
				assert.Equal(t, tt.expectedDefault, fieldInfo.Default)
				assert.NotEmpty(t, fieldInfo.Description)

				if tt.expectedValidation {
					assert.NotNil(t, fieldInfo.Validation, "Expected validation function to be present")
				} else {
					assert.Nil(t, fieldInfo.Validation, "Expected no validation function")
				}
			}
		})
	}
}

func TestFindSimilarKeys(t *testing.T) {
	schema := DefaultConfigSchema()

	tests := []struct {
		name             string
		key              string
		expectedContains []string
		maxSuggestions   int
	}{
		{
			name:             "Alias match - temp should suggest temperature",
			key:              "temp",
			expectedContains: []string{"temperature"},
			maxSuggestions:   5,
		},
		{
			name:             "Canonical key match - token should suggest max_tokens",
			key:              "token",
			expectedContains: []string{"parameters.max_tokens"},
			maxSuggestions:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := schema.FindSimilarKeys(tt.key)

			// check that we don't exceed the maximum suggestions limit
			assert.LessOrEqual(t, len(suggestions), tt.maxSuggestions)

			if len(tt.expectedContains) == 0 {
				// expecting no matches
				assert.Empty(t, suggestions)
			} else {
				// check that all expected strings are contained in suggestions
				for _, expected := range tt.expectedContains {
					found := false
					for _, suggestion := range suggestions {
						if suggestion == expected {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected suggestion %q not found in results: %v", expected, suggestions)
				}
			}

			// verify suggestions limit (should not exceed 5)
			assert.LessOrEqual(t, len(suggestions), 5, "Suggestions should be limited to 5 items")
		})
	}
}

// TestSchemaConsistency verifies that the schema is internally consistent
func TestSchemaConsistency(t *testing.T) {
	schema := DefaultConfigSchema()

	t.Run("All aliases point to valid paths", func(t *testing.T) {
		for alias, canonicalPath := range schema.Aliases {
			_, exists := schema.ValidPaths[canonicalPath]
			assert.True(t, exists, "Alias %q points to non-existent path %q", alias, canonicalPath)
		}
	})

	t.Run("All field infos have types", func(t *testing.T) {
		for path, fieldInfo := range schema.ValidPaths {
			assert.NotNil(t, fieldInfo.Type, "Field %q has nil type", path)
			assert.NotEmpty(t, fieldInfo.Description, "Field %q has empty description", path)
		}
	})

	t.Run("Validation functions work correctly", func(t *testing.T) {
		// test a few key validation functions
		tempInfo := schema.ValidPaths["parameters.temperature"]
		if tempInfo.Validation != nil {
			err := tempInfo.Validation(0.5) // valid value
			assert.NoError(t, err)

			err = tempInfo.Validation(2.0) // invalid value
			assert.Error(t, err)
		}

		maxTokensInfo := schema.ValidPaths["parameters.max_tokens"]
		if maxTokensInfo.Validation != nil {
			err := maxTokensInfo.Validation(1000) // valid value
			assert.NoError(t, err)

			err = maxTokensInfo.Validation(0) // invalid value
			assert.Error(t, err)
		}
	})
}
