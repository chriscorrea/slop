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

func TestDescribeCommand_CanonicalKey(t *testing.T) {
	// setup temp config
	tempDir, err := os.MkdirTemp("", "slop-test-describe-canonical-*")
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

	// set a test value
	manager.Viper().Set("parameters.temperature", 0.8)

	// setup state
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// create command and capture output
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// exec with canonical key
	err = describeConfigCmd.RunE(cmd, []string{"parameters.temperature"})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// verify essential output components
	expectedContent := []string{
		"Configuration Key: parameters.temperature",
		"Type: float64",
		"Description: LLM temperature for response randomness",
		"Current Value: 0.8",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// should NOT show alias line if canonical key input
	if strings.Contains(output, "Alias:") {
		t.Error("Should not show alias line when input is canonical key")
	}
}

func TestDescribeCommand_Alias(t *testing.T) {
	// setup temporary config
	tempDir, err := os.MkdirTemp("", "slop-test-describe-alias-*")
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

	// set a test value
	manager.Viper().Set("parameters.temperature", 0.7)

	// setup state
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// create command and capture output
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// exec with alias
	err = describeConfigCmd.RunE(cmd, []string{"temp"})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// verify essential output components
	expectedContent := []string{
		"Configuration Key: parameters.temperature", // Resolved canonical key
		"Alias: temp", // Shows the alias used
		"Type: float64",
		"Description: LLM temperature for response randomness",
		"Current Value: 0.7",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestDescribeCommand_InvalidKey(t *testing.T) {
	// setup temporary config
	tempDir, err := os.MkdirTemp("", "slop-test-describe-invalid-*")
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

	// create command
	cmd := &cobra.Command{}

	// Execute with invalid key
	err = describeConfigCmd.RunE(cmd, []string{"nonexistent.key"})
	if err == nil {
		t.Fatal("Expected error for invalid key, but got none")
	}

	// verify error message includes the invalid key
	if !strings.Contains(err.Error(), "nonexistent.key") {
		t.Errorf("Expected error message to mention invalid key, got: %v", err)
	}
}

func TestDescribeCommand_InvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no arguments",
			args: []string{},
		},
		{
			name: "too many arguments",
			args: []string{"temp", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary config
			tempDir, err := os.MkdirTemp("", "slop-test-describe-args-*")
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

			// create command
			cmd := &cobra.Command{}

			// test argument validation (handled by Cobra's Args validator)
			err = describeConfigCmd.Args(cmd, tt.args)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
		})
	}
}
