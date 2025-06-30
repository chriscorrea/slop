package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"slop/internal/config"

	"github.com/spf13/cobra"
)

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name          string
		canonicalPath string
		configValue   interface{}
		expected      string
	}{
		{
			name:          "string value",
			canonicalPath: "parameters.system_prompt",
			configValue:   "You are an utmost assistant",
			expected:      "You are an utmost assistant",
		},
		{
			name:          "float value",
			canonicalPath: "parameters.temperature",
			configValue:   0.75,
			expected:      "0.75",
		},
		{
			name:          "int value",
			canonicalPath: "parameters.max_tokens",
			configValue:   479,
			expected:      "479",
		},
		{
			name:          "nil value",
			canonicalPath: "parameters.unknown",
			configValue:   nil,
			expected:      "<not set>",
		},
		{
			name:          "empty string",
			canonicalPath: "parameters.system_prompt",
			configValue:   "",
			expected:      "<not set>",
		},
		{
			name:          "long value truncated",
			canonicalPath: "parameters.system_prompt",
			configValue:   strings.Repeat("p", 50),
			expected:      strings.Repeat("p", 37) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup temp config
			tempDir, err := os.MkdirTemp("", "slop-test-getvalue-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "config.toml")
			manager := config.NewManager()
			err = manager.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// set the test value
			if tt.configValue != nil {
				manager.Viper().Set(tt.canonicalPath, tt.configValue)
			}

			// setup state
			originalState := state
			state = &rootCmdState{manager: manager}
			defer func() { state = originalState }()

			// test the function
			result := getConfigValue(tt.canonicalPath)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "Long API key is masked",
			key:      "anthropic_key",
			value:    "sk-1234567890abcdef",
			expected: "sk-12...",
		},
		{
			name:     "short API key fully masked",
			key:      "api_key",
			value:    "abc",
			expected: "***",
		},
		{
			name:     "regular parameter not masked",
			key:      "temperature",
			value:    "0.7",
			expected: "0.7",
		},
		{
			name:     "not set value",
			key:      "some_key",
			value:    "<not set>",
			expected: "<not set>",
		},
		{
			name:     "empty value",
			key:      "some_key",
			value:    "",
			expected: "<not set>",
		},
		{
			name:     "regular config value",
			key:      "max_tokens",
			value:    "2048",
			expected: "2048",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValue(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMaskSensitiveValueCanonical(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "API key canonical masked",
			key:      "providers.mistral.api_key",
			value:    "mk-1234567890abcdef",
			expected: "mk-12...",
		},
		{
			name:     "all long values truncated",
			key:      "parameters.system_prompt",
			value:    strings.Repeat("x", 50),
			expected: strings.Repeat("x", 37) + "...",
		},
		{
			name:     "short, normal value not truncated",
			key:      "parameters.temperature",
			value:    "0.7",
			expected: "0.7",
		},
		{
			name:     "not set value",
			key:      "some.key",
			value:    "<not set>",
			expected: "<not set>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValueCanonical(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestListCommand_AliasesView(t *testing.T) {
	// setup temporary config
	tempDir, err := os.MkdirTemp("", "slop-test-list-aliases-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")
	manager := config.NewManager()
	err = manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// set some test configuration values
	manager.Viper().Set("parameters.temperature", 0.8)
	manager.Viper().Set("parameters.max_tokens", 4096)
	manager.Viper().Set("providers.anthropic.api_key", "sk-test123")

	// setup state
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// create command and capture output
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// execute with --aliases flag
	err = listConfigCmd.RunE(cmd, []string{"--aliases"})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// verify key sections are present
	if !strings.Contains(output, "Parameters") {
		t.Error("Expected 'Parameters' section in aliases view")
	}
	if !strings.Contains(output, "Providers") {
		t.Error("Expected 'Providers' section in aliases view")
	}

	// verify some expected aliases appear
	expectedAliases := []string{"temp", "max-tokens"}
	for _, alias := range expectedAliases {
		if !strings.Contains(output, alias) {
			t.Errorf("Expected alias '%s' to appear in output", alias)
		}
	}
}

func TestListCommand_CanonicalView(t *testing.T) {
	// setup temporary config
	tempDir, err := os.MkdirTemp("", "slop-test-list-canonical-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")
	manager := config.NewManager()
	err = manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// setup state
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// create command and capture output
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// set --canonical flag
	cmd.Flags().Bool("canonical", false, "Show canonical configuration paths")
	_ = cmd.Flags().Set("canonical", "true")

	// exec with --canonical flag
	err = listConfigCmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// verify canonical paths appear
	expectedPaths := []string{"parameters.temperature", "parameters.max_tokens"}
	for _, path := range expectedPaths {
		if !strings.Contains(output, path) {
			t.Errorf("Expected canonical path '%s' to appear in output", path)
		}
	}

	// verify sections are present
	if !strings.Contains(output, "Parameters") {
		t.Error("Expected 'Parameters' section in canonical view")
	}
}

func TestListCommand_DefaultBehavior(t *testing.T) {
	// setup temporary config
	tempDir, err := os.MkdirTemp("", "slop-test-list-default-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")
	manager := config.NewManager()
	err = manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// setup state
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// create command and capture output
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// execute without flags–should default to aliases view
	err = listConfigCmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// should behave like aliases view by default
	if !strings.Contains(output, "Parameters") {
		t.Error("Expected 'Parameters' section in default view")
	}

	// should contain alias-style content, not canonical paths
	expectedAliases := []string{"temp"}
	for _, alias := range expectedAliases {
		if !strings.Contains(output, alias) {
			t.Errorf("Expected alias '%s' to appear in default output", alias)
		}
	}
}
