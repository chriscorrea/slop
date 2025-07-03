package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/spf13/cobra"
)

func TestSetCommand_Integration(t *testing.T) {
	// create temporary directory for test
	tempDir, err := os.MkdirTemp("", "slop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")

	// table-driven tests for set
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		checkValue    func(*testing.T, *config.Manager)
	}{
		{
			name: "set temperature",
			args: []string{"parameters.temperature=0.9"},
			checkValue: func(t *testing.T, manager *config.Manager) {
				cfg := manager.Config()
				if cfg.Parameters.Temperature != 0.9 {
					t.Errorf("Expected temperature 0.9, got %f", cfg.Parameters.Temperature)
				}
			},
		},
		{
			name: "set system prompt",
			args: []string{`parameters.system_prompt="You are a specialized assistant"`},
			checkValue: func(t *testing.T, manager *config.Manager) {
				cfg := manager.Config()
				expected := "You are a specialized assistant"
				if cfg.Parameters.SystemPrompt != expected {
					t.Errorf("Expected system prompt %q, got %q", expected, cfg.Parameters.SystemPrompt)
				}
			},
		},
		{
			name: "set max tokens",
			args: []string{"parameters.max_tokens=4096"},
			checkValue: func(t *testing.T, manager *config.Manager) {
				cfg := manager.Config()
				if cfg.Parameters.MaxTokens != 4096 {
					t.Errorf("Expected max_tokens 4096, got %d", cfg.Parameters.MaxTokens)
				}
			},
		},
		{
			name: "set nested model provider",
			args: []string{"models.remote.fast.provider=openai"},
			checkValue: func(t *testing.T, manager *config.Manager) {
				cfg := manager.Config()
				expected := "openai"
				if cfg.Models.Remote.Fast.Provider != expected {
					t.Errorf("Expected provider %q, got %q", expected, cfg.Models.Remote.Fast.Provider)
				}
			},
		},
		{
			name:          "invalid format missing equals",
			args:          []string{"parameters.temperature"},
			expectError:   true,
			errorContains: "invalid format: expected key=value",
		},
		{
			name:        "valid format multiple equals",
			args:        []string{"parameters.system_prompt=Be helpful and concise"},
			expectError: false, // This should work, value becomes "Be helpful and concise"
			checkValue: func(t *testing.T, manager *config.Manager) {
				// check that the value is properly set to "Be helpful and concise"
				cfg := manager.Config()
				expected := "Be helpful and concise"
				if cfg.Parameters.SystemPrompt != expected {
					t.Errorf("Expected system prompt to be %q, got %q", expected, cfg.Parameters.SystemPrompt)
				}
			},
		},
		{
			name:          "empty key",
			args:          []string{"=value"},
			expectError:   true,
			errorContains: "key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh config manager for each test
			manager := config.NewManager()

			// Load initial configuration
			err := manager.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// set up the command state
			originalState := state
			state = &rootCmdState{manager: manager}
			defer func() { state = originalState }()

			// create command and capture output
			cmd := &cobra.Command{}
			cmd.SetArgs(tt.args)

			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// execute the set command
			err = setCmd.RunE(cmd, tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// verify the configuration was saved via reload
			if tt.checkValue != nil {
				freshManager := config.NewManager()
				err = freshManager.Load(configPath)
				if err != nil {
					t.Fatalf("Failed to reload config: %v", err)
				}
				tt.checkValue(t, freshManager)
			}

			// verify output message
			output := stdout.String()
			if !strings.Contains(output, "Configuration updated:") {
				t.Errorf("Expected success message in output, got: %q", output)
			}
		})
	}
}

func TestSetCommand_Args(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"no args", []string{}, true},
		{"one arg", []string{"key=value"}, false},
		{"two args", []string{"key=value", "extra"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			err := setCmd.Args(cmd, tt.args)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSetCommand_PersistenceBetweenSessions(t *testing.T) {
	// ceate temporary directory for test
	tempDir, err := os.MkdirTemp("", "slop-persistence-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")

	// first session: set a value
	{
		manager := config.NewManager()
		err := manager.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		state = &rootCmdState{manager: manager}

		cmd := &cobra.Command{}
		err = setCmd.RunE(cmd, []string{"parameters.temperature=0.8"})
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}
	}

	// second session: verify value persisted
	{
		manager := config.NewManager()
		err := manager.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config in second session: %v", err)
		}

		cfg := manager.Config()
		if cfg.Parameters.Temperature != 0.8 {
			t.Errorf("Value did not persist between sessions. Expected 0.8, got %f", cfg.Parameters.Temperature)
		}
	}

	// third session: set another value, verify both persist
	{
		manager := config.NewManager()
		err := manager.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config in third session: %v", err)
		}

		state = &rootCmdState{manager: manager}

		cmd := &cobra.Command{}
		err = setCmd.RunE(cmd, []string{"parameters.max_tokens=8192"})
		if err != nil {
			t.Fatalf("Failed to set second value: %v", err)
		}

		// Verify both values
		cfg := manager.Config()
		if cfg.Parameters.Temperature != 0.8 {
			t.Errorf("First value lost. Expected temperature 0.8, got %f", cfg.Parameters.Temperature)
		}
		if cfg.Parameters.MaxTokens != 8192 {
			t.Errorf("Second value not set. Expected max_tokens 8192, got %d", cfg.Parameters.MaxTokens)
		}
	}
}

func TestConvertValueToType(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		targetType  reflect.Type
		expected    interface{}
		expectError bool
	}{
		// table-driven string type tests
		{
			name:       "unquoted string",
			value:      "Hear the hoofbeats of tomorrow",
			targetType: reflect.TypeOf(""),
			expected:   "Hear the hoofbeats of tomorrow",
		},
		{
			name:       "double quoted string",
			value:      `"See the golden future rise!"`,
			targetType: reflect.TypeOf(""),
			expected:   "See the golden future rise!",
		},
		{
			name:       "single quoted multi-char string (invalid, fallback to original)",
			value:      `'All animals are equal'`,
			targetType: reflect.TypeOf(""),
			expected:   `'All animals are equal'`, // keep original
		},
		{
			name:       "single quoted single char",
			value:      `'n'`,
			targetType: reflect.TypeOf(""),
			expected:   "n",
		},
		{
			name:       "backtick quoted string",
			value:      "`Some animals are more equal than others`",
			targetType: reflect.TypeOf(""),
			expected:   "Some animals are more equal than others",
		},
		{
			name:       "string with escape sequences",
			value:      `"oink\noink\ttab"`,
			targetType: reflect.TypeOf(""),
			expected:   "oink\noink\ttab",
		},
		{
			name:       "string with escaped quotes",
			value:      `"to \"Sugarcandy\" Mountain"`,
			targetType: reflect.TypeOf(""),
			expected:   `to "Sugarcandy" Mountain`,
		},
		{
			name:       "empty quoted string",
			value:      `""`,
			targetType: reflect.TypeOf(""),
			expected:   "",
		},
		{
			name:       "string that starts with quote but isn't quoted",
			value:      `"oink oink`,
			targetType: reflect.TypeOf(""),
			expected:   `"oink oink`,
		},

		// table-driven boolean type tests
		{
			name:       "boolean true unquoted",
			value:      "true",
			targetType: reflect.TypeOf(true),
			expected:   true,
		},
		{
			name:       "boolean false quoted",
			value:      `"false"`,
			targetType: reflect.TypeOf(true),
			expected:   false,
		},
		{
			name:       "boolean TRUE case insensitive",
			value:      "TRUE",
			targetType: reflect.TypeOf(true),
			expected:   true,
		},
		{
			name:        "invalid boolean",
			value:       "not_a_bool",
			targetType:  reflect.TypeOf(true),
			expectError: true,
		},

		// table-driven int type tests
		{
			name:       "integer unquoted",
			value:      "55",
			targetType: reflect.TypeOf(int(0)),
			expected:   55,
		},
		{
			name:       "integer quoted",
			value:      `"123"`,
			targetType: reflect.TypeOf(int(0)),
			expected:   123,
		},
		{
			name:       "negative integer",
			value:      "-499",
			targetType: reflect.TypeOf(int(0)),
			expected:   -499,
		},
		{
			name:        "invalid integer",
			value:       "not_a_number",
			targetType:  reflect.TypeOf(int(0)),
			expectError: true,
		},
		// table-driven float64 type tests
		{
			name:       "float64 decimal",
			value:      "3.1415",
			targetType: reflect.TypeOf(float64(0)),
			expected:   3.1415,
		},
		{
			name:       "float64 quoted",
			value:      `"2.7182"`,
			targetType: reflect.TypeOf(float64(0)),
			expected:   2.7182,
		},
		{
			name:       "float64 integer value",
			value:      "42",
			targetType: reflect.TypeOf(float64(0)),
			expected:   float64(42),
		},
		{
			name:        "invalid float64",
			value:       "not_a_float",
			targetType:  reflect.TypeOf(float64(0)),
			expectError: true,
		},

		// table-driven Float32 type tests
		{
			name:       "float32 value",
			value:      "9.5",
			targetType: reflect.TypeOf(float32(0)),
			expected:   float32(9.5),
		},
		{
			name:       "float32 quoted",
			value:      `"0.5"`,
			targetType: reflect.TypeOf(float32(0)),
			expected:   float32(0.5),
		},

		// Unsupported type test
		{
			name:        "unsupported type",
			value:       "some_value",
			targetType:  reflect.TypeOf([]string{}),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValueToType(tt.value, tt.targetType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}
