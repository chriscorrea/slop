package cmd

import (
	"testing"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSelectModel(t *testing.T) {
	// create a base configuration from embedded defaults.
	// note that remote/fast is default in embedded config (NewDefaultFromEmbedded)
	fullConfig := config.NewDefaultFromEmbedded()

	// fictional models for testing
	fullConfig.Models.Remote.Fast.Provider = "mistral"
	fullConfig.Models.Remote.Fast.Name = "mistral-fast-1"
	fullConfig.Models.Remote.Deep.Provider = "mistral"
	fullConfig.Models.Remote.Deep.Name = "mistral-deep-9"

	// set a dummy API key to satisfy configuration validation checks.
	fullConfig.Providers.Mistral.APIKey = "test-mistral-key"

	// define a  named command
	fullConfig.Commands["sestina"] = config.Command{
		Description: "A test named command using a remote, fast model.",
		ModelType:   "remote-fast",
	}

	// define a named command with specific model type
	fullConfig.Commands["villanelle"] = config.Command{
		Description: "A test named command using a local, deep model.",
		Temperature: func() *float64 { f := 0.7; return &f }(),
		ModelType:   "local-deep",
	}

	// table-driven tests for model selection
	tests := []struct {
		name             string
		args             []string
		flagSettings     map[string]string
		expectedProvider string
		expectedModel    string
		expectError      bool
	}{
		// basic flags
		{
			name:             "Default (no flags)",
			args:             []string{},
			flagSettings:     nil,
			expectedProvider: "mistral",
			expectedModel:    "mistral-fast-1",
		},
		{
			name:             "Remote Deep model via --deep flag",
			args:             []string{},
			flagSettings:     map[string]string{"deep": "true"},
			expectedProvider: "mistral",
			expectedModel:    "mistral-deep-9",
		},
		{
			name:             "Local Fast model via --local flag",
			args:             []string{},
			flagSettings:     map[string]string{"local": "true"},
			expectedProvider: "ollama",
			expectedModel:    "gemma3:latest", // from default_config.toml
		},
		{
			name:             "Local Deep model via --local and --deep flags",
			args:             []string{},
			flagSettings:     map[string]string{"local": "true", "deep": "true"},
			expectedProvider: "ollama",
			expectedModel:    "deepseek-r1:14b", // from default_config.toml
		},
		{
			name:             "Test mode flag overrides all other flags",
			args:             []string{},
			flagSettings:     map[string]string{"test": "true", "local": "true", "deep": "true"},
			expectedProvider: "mock",
			expectedModel:    "test-model",
		},
		// named command presets
		{
			name:             "Named command with remote fast model",
			args:             []string{"sestina", "a sample prompt"},
			flagSettings:     nil, // no flags set, so named command presets should be used
			expectedProvider: "mistral",
			expectedModel:    "mistral-fast-1",
		},
		{
			name:             "Named command proceeded by invalid junk",
			args:             []string{"--local=xyz", "sestina", "a sample prompt"},
			flagSettings:     nil, // no valid flags set, so named command presets should be used
			expectedProvider: "mistral",
			expectedModel:    "mistral-fast-1",
		},
		{
			name:             "Named command hints are applied correctly",
			args:             []string{"villanelle", "a sample prompt"},
			flagSettings:     nil, // no flags set, so named command presets should be used
			expectedProvider: "ollama",
			expectedModel:    "deepseek-r1:14b",
		},
		{
			name:             "Explicit --remote flag overrides named command's local preset only",
			args:             []string{"villanelle", "a sample prompt"},
			flagSettings:     map[string]string{"local": "false"},
			expectedProvider: "mistral",
			expectedModel:    "mistral-deep-9", // should maintain deep model
		},
		{
			name:             "Explicit --fast flag overrides named command deep preset only",
			args:             []string{"villanelle", "a sample prompt"},
			flagSettings:     map[string]string{"deep": "false"}, // local preset still applies.
			expectedProvider: "ollama",
			expectedModel:    "gemma3:latest",
		},
		{
			name:             "Multiple flags can completely override named command presets",
			args:             []string{"villanelle", "a sample prompt"},
			flagSettings:     map[string]string{"local": "true", "deep": "true"},
			expectedProvider: "ollama",
			expectedModel:    "deepseek-r1:14b",
		},

		// No flags
		// [note that remote/fast is default in embedded config (see NewDefaultFromEmbedded)]
		{
			name:             "Direct prompt (no named command) does not trigger presets",
			args:             []string{"a prompt without a command"}, // no defined command name
			flagSettings:     nil,
			expectedProvider: "mistral",
			expectedModel:    "mistral-fast-1",
		},
	}

	// test execution loop
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create a fresh command and selector for each test case.
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test", false, "Test mode")
			cmd.Flags().Bool("local", false, "Local mode")
			cmd.Flags().Bool("deep", false, "Deep mode")
			cmd.Flags().Bool("remote", false, "Remote mode")
			cmd.Flags().Bool("fast", false, "Fast mode")
			selector := NewModelSelector()

			// Apply flag settings for the current test case.
			if tt.flagSettings != nil {
				for flag, value := range tt.flagSettings {
					// Mark the flag as changed to simulate user input.
					err := cmd.Flags().Set(flag, value)
					assert.NoError(t, err, "Failed to set flag %s=%s", flag, value)
				}
			}

			// Execute the model selection logic.
			provider, model, err := selector.SelectModel(cmd, fullConfig, tt.args)

			// Assert the results.
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedProvider, provider, "Provider name did not match expected value.")
				assert.Equal(t, tt.expectedModel, model, "Model name did not match expected value.")
			}
		})
	}
}

func TestDefaultModelSelector_SelectModel(t *testing.T) {
	cfg := config.NewDefaultFromEmbedded()
	// set API keys to avoid validation errors
	cfg.Providers.Mistral.APIKey = "test-mistral-key"
	selector := NewModelSelector()

	tests := []struct {
		name             string
		flagSettings     map[string]string
		expectedProvider string
		expectedModel    string
	}{
		{
			name:             "test mode",
			flagSettings:     map[string]string{"test": "true"},
			expectedProvider: "mock",
			expectedModel:    "test-model",
		},
		{
			name:             "local flag",
			flagSettings:     map[string]string{"local": "true"},
			expectedProvider: "ollama",
			expectedModel:    "gemma3:latest",
		},
		{
			name:             "local + deep",
			flagSettings:     map[string]string{"local": "true", "deep": "true"},
			expectedProvider: "ollama",
			expectedModel:    "deepseek-r1:14b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command with flags
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test", false, "Test mode")
			cmd.Flags().Bool("local", false, "Local mode")
			cmd.Flags().Bool("fast", false, "Fast mode")
			cmd.Flags().Bool("heavy", false, "Heavy mode")
			cmd.Flags().Bool("deep", false, "Deep mode")

			// Set flags based on test case
			for flag, value := range tt.flagSettings {
				err := cmd.Flags().Set(flag, value)
				if err != nil {
					t.Fatalf("Failed to set flag %s=%s: %v", flag, value, err)
				}
			}

			provider, model, err := selector.SelectModel(cmd, cfg, []string{})
			if err != nil {
				t.Fatalf("SelectModel failed: %v", err)
			}

			if provider != tt.expectedProvider {
				t.Errorf("Expected provider %q, got %q", tt.expectedProvider, provider)
			}

			if model != tt.expectedModel {
				t.Errorf("Expected model %q, got %q", tt.expectedModel, model)
			}
		})
	}
}
