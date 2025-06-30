package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestNewDefault(t *testing.T) {
	config := NewDefaultFromEmbedded()

	// config is not nil
	if config == nil {
		t.Fatal("NewDefault() returned nil")
	}

	t.Run("Parameters", func(t *testing.T) {
		parameters := config.Parameters

		if parameters.Temperature != 0.7 {
			t.Errorf("Expected Temperature to be 0.7, got %f", parameters.Temperature)
		}

		if parameters.MaxTokens != 2048 {
			t.Errorf("Expected MaxTokens to be 2048, got %d", parameters.MaxTokens)
		}

		if parameters.TopP != 1.0 {
			t.Errorf("Expected TopP to be 1.0, got %f", parameters.TopP)
		}

		expectedStopSequences := []string{"\n\n", "STOP", "END"}
		if !reflect.DeepEqual(parameters.StopSequences, expectedStopSequences) {
			t.Errorf("Expected StopSequences to be %v, got %v", expectedStopSequences, parameters.StopSequences)
		}

		if !strings.Contains(parameters.SystemPrompt, "You are a helpful") {
			t.Errorf("Expected SystemPrompt to contain 'You are a helpful', got %s", parameters.SystemPrompt)
		}

		if parameters.Timeout != 30 {
			t.Errorf("Expected Timeout to be 30, got %d", parameters.Timeout)
		}
	})

	// remote models
	t.Run("RemoteModels", func(t *testing.T) {
		remote := config.Models.Remote

		// fast model - just ensure it's properly configured
		if remote.Fast.Provider == "" {
			t.Errorf("Expected Remote Fast Provider to be configured")
		}

		if remote.Fast.Name == "" {
			t.Errorf("Expected Remote Fast Name to be configured")
		}

		// deep model - just ensure it's properly configured
		if remote.Deep.Provider == "" {
			t.Errorf("Expected Remote Deep Provider to be configured")
		}

		if remote.Deep.Name == "" {
			t.Errorf("Expected Remote Deep Name to be configured")
		}
	})

	// local models
	t.Run("LocalModels", func(t *testing.T) {
		local := config.Models.Local

		// Test Fast model
		if local.Fast.Provider != "ollama" {
			t.Errorf("Expected Local Fast Provider to be 'ollama', got %s", local.Fast.Provider)
		}

		if local.Fast.Name != "gemma3:latest" {
			t.Errorf("Expected Local Fast Name to be 'gemma3:latest', got %s", local.Fast.Name)
		}

		// Test Deep model
		if local.Deep.Provider != "ollama" {
			t.Errorf("Expected Local Deep Provider to be 'ollama', got %s", local.Deep.Provider)
		}

		if local.Deep.Name != "deepseek-r1:14b" {
			t.Errorf("Expected Local Deep Name to be 'deepseek-r1:14b', got %s", local.Deep.Name)
		}
	})
}

// TestConfigStructure verifies the config structure is well defined
func TestConfigStructure(t *testing.T) {
	config := NewDefaultFromEmbedded()

	// all major sections exist
	if config.Parameters.Temperature == 0 && config.Parameters.MaxTokens == 0 {
		t.Error("Parameters section appears to be uninitialized")
	}

	if config.Models.Remote.Fast.Provider == "" {
		t.Error("Remote models section appears to be uninitialized")
	}

	if config.Models.Local.Fast.Provider == "" {
		t.Error("Local models section appears to be uninitialized")
	}

	if config.Providers.Anthropic.BaseUrl == "" {
		t.Error("Providers section appears to be uninitialized")
	}
}

// TestStopSequencesNotNil ensures stop sequences slice is initialized
func TestStopSequencesNotNil(t *testing.T) {
	config := NewDefaultFromEmbedded()

	if config.Parameters.StopSequences == nil {
		t.Error("StopSequences should not be nil")
	}

	if len(config.Parameters.StopSequences) == 0 {
		t.Error("StopSequences should have default values")
	}
}

// TestProviderBaseUrls verifies that providers have base URLs
func TestProviderBaseUrls(t *testing.T) {
	config := NewDefaultFromEmbedded()

	tests := []struct {
		name     string
		provider string
		baseUrl  string
	}{
		{"Anthropic", "anthropic", config.Providers.Anthropic.BaseUrl},
		{"OpenAI", "openai", config.Providers.OpenAI.BaseUrl},
		{"Ollama", "ollama", config.Providers.Ollama.BaseUrl},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.baseUrl == "" && tt.provider != "cohere" {
				t.Errorf("Provider %s should have a non-empty BaseUrl", tt.provider)
			}
		})
	}
}

// TestLoad tests the configuration manager's Load method
func TestLoad(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(t *testing.T, tempDir string) string
		expectError    bool
		errorType      string
		validateConfig func(t *testing.T, cfg *Config)
	}{
		{
			name: "Successful Load",
			setupFile: func(t *testing.T, tempDir string) string {
				configPath := filepath.Join(tempDir, "test_config.toml")
				configContent := `[parameters]
temperature = 0.5
system_prompt = "Custom prompt"
max_tokens = 1024

[models.remote.fast]
provider = "entropic"
name = "sestina-4-1"
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				return configPath
			},
			expectError: false,
			validateConfig: func(t *testing.T, cfg *Config) {
				if cfg.Parameters.Temperature != 0.5 {
					t.Errorf("Expected Temperature to be 0.5, got %f", cfg.Parameters.Temperature)
				}
				if cfg.Parameters.SystemPrompt != "Custom prompt" {
					t.Errorf("Expected SystemPrompt to be 'Custom prompt', got %s", cfg.Parameters.SystemPrompt)
				}
				if cfg.Parameters.MaxTokens != 1024 {
					t.Errorf("Expected MaxTokens to be 1024, got %d", cfg.Parameters.MaxTokens)
				}
				if cfg.Models.Remote.Fast.Provider != "entropic" {
					t.Errorf("Expected Fast Provider to be 'entropic', got %s", cfg.Models.Remote.Fast.Provider)
				}
				if cfg.Models.Remote.Fast.Name != "sestina-4-1" {
					t.Errorf("Expected Fast Name to be 'sestina-4-1', got %s", cfg.Models.Remote.Fast.Name)
				}
				// verify defaults are preserved for non-overridden values
				if cfg.Parameters.TopP != 1.0 {
					t.Errorf("Expected TopP default to be preserved as 1.0, got %f", cfg.Parameters.TopP)
				}
			},
		},
		{
			name: "File Not Found",
			setupFile: func(t *testing.T, tempDir string) string {
				return filepath.Join(tempDir, "nonexistent.toml")
			},
			expectError: false,
			validateConfig: func(t *testing.T, cfg *Config) {
				// should contain all default values
				if cfg.Parameters.Temperature != 0.7 {
					t.Errorf("Expected default Temperature to be 0.7, got %f", cfg.Parameters.Temperature)
				}
				if !strings.Contains(cfg.Parameters.SystemPrompt, "You are a helpful") {
					t.Errorf("Expected SystemPrompt to contain 'You are a helpful', got %s", cfg.Parameters.SystemPrompt)
				}
				if cfg.Models.Remote.Fast.Provider == "" {
					t.Errorf("Expected default Fast Provider to be configured, got empty string")
				}
			},
		},
		{
			name: "Malformed File",
			setupFile: func(t *testing.T, tempDir string) string {
				configPath := filepath.Join(tempDir, "malformed.toml")
				configContent := `[parameters
temperature = "invalid
missing_quote = test`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create malformed config file: %v", err)
				}
				return configPath
			},
			expectError: true,
			errorType:   "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := tt.setupFile(t, tempDir)

			manager := NewManager()
			err := manager.Load(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				// for malformed files, expect a parse error
				if tt.errorType == "parse" {
					// verify it's not a ConfigFileNotFoundError
					var configFileNotFoundError viper.ConfigFileNotFoundError
					if errors.As(err, &configFileNotFoundError) {
						t.Errorf("Expected parse error, but got ConfigFileNotFoundError")
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if tt.validateConfig != nil {
					tt.validateConfig(t, manager.Config())
				}
			}
		})
	}
}

// TestNewManager tests the manager constructor
func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.v == nil {
		t.Error("Manager's Viper instance is nil")
	}

	if manager.cfg == nil {
		t.Error("Manager's Config instance is nil")
	}

	// verify config starts empty (defaults loaded in Load())
	cfg := manager.Config()
	if cfg.Parameters.Temperature != 0.0 {
		t.Errorf("Expected empty Temperature to be 0.0, got %f", cfg.Parameters.Temperature)
	}
}

// TestDefaultCommands verifies that default commands are properly loaded
func TestDefaultCommands(t *testing.T) {
	config := NewDefaultFromEmbedded()

	// verify default commands exist
	expectedCommands := []string{"compress", "expand", "explain"}
	for _, cmdName := range expectedCommands {
		if command, exists := config.Commands[cmdName]; !exists {
			t.Errorf("Expected default command '%s' to exist", cmdName)
		} else {
			if command.Description == "" {
				t.Errorf("Expected command '%s' to have a description", cmdName)
			}
			if command.SystemPrompt == "" {
				t.Errorf("Expected command '%s' to have a system prompt", cmdName)
			}
		}
	}
}

// TestLoadCommandsFile tests loading commands from commands.toml
func TestLoadCommandsFile(t *testing.T) {
	tests := []struct {
		name             string
		commandsContent  string
		expectedCommands map[string]string // command name -> description
		expectError      bool
		errorContains    string
	}{
		{
			name:            "Default commands only",
			commandsContent: "", // no commands.toml
			expectedCommands: map[string]string{
				"compress": "Summarize and compress long text concisely",
				"expand":   "Elaborate and expand on brief text with rich detail",
			},
			expectError: false,
		},
		{
			name: "User commands extend defaults",
			commandsContent: `
[commands]

[commands.summarize]
description = "Custom summarization command"
system_prompt = "You are a summarization expert."
reasoning = true

[commands.translate]
description = "Translate text between languages"
system_prompt = "You are a professional translator."
temperature = 0.3
`,
			expectedCommands: map[string]string{
				"compress":  "Summarize and compress long text concisely", // Default
				"summarize": "Custom summarization command",               // User
				"translate": "Translate text between languages",           // User
			},
			expectError: false,
		},
		{
			name: "User command overrides default",
			commandsContent: `
[commands]

[commands.compress]
description = "Custom compression"
system_prompt = "Custom compress prompt"
temperature = 0.2
`,
			expectedCommands: map[string]string{
				"compress": "Custom compression",                                  // Overridden
				"expand":   "Elaborate and expand on brief text with rich detail", // Default preserved
			},
			expectError: false,
		},
		{
			name: "Reserved command override fails",
			commandsContent: `
[commands]

[commands.help]
description = "Custom help"
system_prompt = "I am custom help"
`,
			expectedCommands: map[string]string{},
			expectError:      true,
			errorContains:    "cannot override reserved command: help",
		},
		{
			name: "Invalid TOML syntax",
			commandsContent: `
[commands.invalid
description = "Missing closing bracket"
`,
			expectedCommands: map[string]string{},
			expectError:      false, // loadUserCommands handles this gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create temp directory and files
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.toml")
			commandsPath := filepath.Join(tempDir, "commands.toml")

			// create minimal config.toml
			configContent := `
[parameters]
temperature = 0.7
system_prompt = "Test prompt"
`
			err := os.WriteFile(configPath, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// create commands.toml if content provided
			if tt.commandsContent != "" {
				err = os.WriteFile(commandsPath, []byte(tt.commandsContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create commands file: %v", err)
				}
			}

			// create manager and load config
			manager := NewManager()
			err = manager.Load(configPath)

			// check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
				return
			}

			// verify expected commands
			cfg := manager.Config()
			for cmdName, expectedDesc := range tt.expectedCommands {
				if command, exists := cfg.Commands[cmdName]; !exists {
					t.Errorf("Expected command '%s' to exist", cmdName)
				} else if command.Description != expectedDesc {
					t.Errorf("Expected command '%s' description to be %q, got %q",
						cmdName, expectedDesc, command.Description)
				}
			}
		})
	}
}

// TestCommandOverrides tests the WithCommandOverrides method
func TestCommandOverrides(t *testing.T) {
	baseConfig := NewDefaultFromEmbedded()
	baseConfig.Parameters.SystemPrompt = "Base prompt"
	baseConfig.Parameters.Temperature = 0.7
	baseConfig.Parameters.MaxTokens = 2048

	command := Command{
		SystemPrompt: "Override prompt",
		Temperature:  &[]float64{0.9}[0],
		// MaxTokens not specified, should use base value
	}

	newConfig := baseConfig.WithCommandOverrides(command)

	// verify overrides were applied
	if newConfig.Parameters.SystemPrompt != "Override prompt" {
		t.Errorf("Expected SystemPrompt to be 'Override prompt', got %q", newConfig.Parameters.SystemPrompt)
	}

	if newConfig.Parameters.Temperature != 0.9 {
		t.Errorf("Expected Temperature to be 0.9, got %f", newConfig.Parameters.Temperature)
	}

	// verify base value preserved when not overridden
	if newConfig.Parameters.MaxTokens != 2048 {
		t.Errorf("Expected MaxTokens to be 2048, got %d", newConfig.Parameters.MaxTokens)
	}

	// verify original config unchanged (no mutation)
	if baseConfig.Parameters.SystemPrompt != "Base prompt" {
		t.Errorf("Original config was mutated")
	}
}

func TestLoadProviderKeysFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		configAPIKey   string
		envAPIKey      string
		envVarName     string
		expectedAPIKey string
		description    string
	}{
		{
			name:           "Load Mistral API key from env when config is empty",
			configAPIKey:   "",
			envAPIKey:      "test-mistral-key",
			envVarName:     "MISTRAL_API_KEY",
			expectedAPIKey: "test-mistral-key",
			description:    "Environment variable should be used when config is empty",
		},
		{
			name:           "Use config API key when env is empty",
			configAPIKey:   "config-mistral-key",
			envAPIKey:      "",
			envVarName:     "MISTRAL_API_KEY",
			expectedAPIKey: "config-mistral-key",
			description:    "Config file value should be used when env is empty",
		},
		{
			name:           "Environment variable overrides config (Viper precedence)",
			configAPIKey:   "config-mistral-key",
			envAPIKey:      "env-mistral-key",
			envVarName:     "MISTRAL_API_KEY",
			expectedAPIKey: "env-mistral-key",
			description:    "Environment variable takes precedence over config file (Viper's standard precedence)",
		},
		{
			name:           "Load Cohere API key from env",
			configAPIKey:   "",
			envAPIKey:      "test-cohere-key",
			envVarName:     "COHERE_API_KEY",
			expectedAPIKey: "test-cohere-key",
			description:    "Cohere environment variable should be loaded",
		},
		{
			name:           "Environment variable overrides Cohere config",
			configAPIKey:   "config-cohere-key",
			envAPIKey:      "env-cohere-key",
			envVarName:     "COHERE_API_KEY",
			expectedAPIKey: "env-cohere-key",
			description:    "Environment variable takes precedence over config file for Cohere",
		},
		{
			name:           "Load OpenAI API key from env",
			configAPIKey:   "",
			envAPIKey:      "test-openai-key",
			envVarName:     "OPENAI_API_KEY",
			expectedAPIKey: "test-openai-key",
			description:    "OpenAI environment variable should be loaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clean environment
			os.Unsetenv(tt.envVarName)

			// set environment variable if specified
			if tt.envAPIKey != "" {
				os.Setenv(tt.envVarName, tt.envAPIKey)
				defer os.Unsetenv(tt.envVarName)
			}

			// create a temporary config file with the config API key
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.toml")

			var configContent string
			if tt.configAPIKey != "" {
				switch tt.envVarName {
				case "MISTRAL_API_KEY":
					configContent = `[providers.mistral]
api_key = "` + tt.configAPIKey + `"`
				case "COHERE_API_KEY":
					configContent = `[providers.cohere]
api_key = "` + tt.configAPIKey + `"`
				case "OPENAI_API_KEY":
					configContent = `[providers.openai]
api_key = "` + tt.configAPIKey + `"`
				}
			}

			if configContent != "" {
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
			}

			// create manager and load config (this will apply Viper's environment variable binding)
			manager := NewManager()
			err := manager.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Check the result
			var actualAPIKey string
			switch tt.envVarName {
			case "MISTRAL_API_KEY":
				actualAPIKey = manager.cfg.Providers.Mistral.APIKey
			case "COHERE_API_KEY":
				actualAPIKey = manager.cfg.Providers.Cohere.APIKey
			case "OPENAI_API_KEY":
				actualAPIKey = manager.cfg.Providers.OpenAI.APIKey
			}

			if actualAPIKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'. %s", tt.expectedAPIKey, actualAPIKey, tt.description)
			}
		})
	}
}

func TestLoadWithEnvFallback(t *testing.T) {
	// create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// create a minimal config file without API keys
	configContent := `[parameters]
temperature = 0.5

[providers.mistral]
api_key = ""

[providers.cohere]
api_key = ""

[providers.openai]
api_key = ""
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables
	os.Setenv("MISTRAL_API_KEY", "env-mistral-key")
	os.Setenv("COHERE_API_KEY", "env-cohere-key")
	os.Setenv("OPENAI_API_KEY", "env-openai-key")
	defer func() {
		os.Unsetenv("MISTRAL_API_KEY")
		os.Unsetenv("COHERE_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	// load config
	manager := NewManager()
	err := manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// verify environment variables were loaded
	if manager.cfg.Providers.Mistral.APIKey != "env-mistral-key" {
		t.Errorf("Expected Mistral API key 'env-mistral-key', got '%s'", manager.cfg.Providers.Mistral.APIKey)
	}

	if manager.cfg.Providers.Cohere.APIKey != "env-cohere-key" {
		t.Errorf("Expected Cohere API key 'env-cohere-key', got '%s'", manager.cfg.Providers.Cohere.APIKey)
	}

	if manager.cfg.Providers.OpenAI.APIKey != "env-openai-key" {
		t.Errorf("Expected OpenAI API key 'env-openai-key', got '%s'", manager.cfg.Providers.OpenAI.APIKey)
	}

	// verify other config vals are still loaded correctly
	if manager.cfg.Parameters.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", manager.cfg.Parameters.Temperature)
	}
}

func TestLoadWithViperEnvPrecedence(t *testing.T) {
	// create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// create a config file with API keys already set
	configContent := `[parameters]
temperature = 0.577

[providers.mistral]
api_key = "config-mistral-key"

[providers.cohere]
api_key = "config-cohere-key"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables (these SHOULD override config values per Viper's precedence)
	os.Setenv("MISTRAL_API_KEY", "env-mistral-key")
	os.Setenv("COHERE_API_KEY", "env-cohere-key")
	defer func() {
		os.Unsetenv("MISTRAL_API_KEY")
		os.Unsetenv("COHERE_API_KEY")
	}()

	// load config
	manager := NewManager()
	err := manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// verify environment variables take precedence over config values (Viper's standard precedence)
	if manager.cfg.Providers.Mistral.APIKey != "env-mistral-key" {
		t.Errorf("Expected Mistral API key 'env-mistral-key', got '%s'", manager.cfg.Providers.Mistral.APIKey)
	}

	if manager.cfg.Providers.Cohere.APIKey != "env-cohere-key" {
		t.Errorf("Expected Cohere API key 'env-cohere-key', got '%s'", manager.cfg.Providers.Cohere.APIKey)
	}

	// verify other config values are still loaded correctly (no env var set for temperature)
	if manager.cfg.Parameters.Temperature != 0.577 {
		t.Errorf("Expected temperature 0.577, got %f", manager.cfg.Parameters.Temperature)
	}
}

// TestCanonicalConfigPaths verifies that API keys are correctly loaded
func TestCanonicalConfigPaths(t *testing.T) {
	tests := []struct {
		name          string
		envVar        string
		envValue      string
		getAPIKey     func(*Manager) string
		canonicalPath string
	}{
		{
			name:          "Mistral canonical path",
			envVar:        "MISTRAL_API_KEY",
			envValue:      "test-mistral-key",
			getAPIKey:     func(m *Manager) string { return m.cfg.Providers.Mistral.APIKey },
			canonicalPath: "providers.mistral.api_key",
		},
		{
			name:          "Cohere canonical path",
			envVar:        "COHERE_API_KEY",
			envValue:      "test-cohere-key",
			getAPIKey:     func(m *Manager) string { return m.cfg.Providers.Cohere.APIKey },
			canonicalPath: "providers.cohere.api_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clean environment
			os.Unsetenv(tt.envVar)

			// set env var
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			// create manager and load config (no config file)
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.toml")

			manager := NewManager()
			err := manager.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// verify API key was loaded from environment
			actualAPIKey := tt.getAPIKey(manager)
			if actualAPIKey != tt.envValue {
				t.Errorf("Expected API key '%s', got '%s'.", tt.envValue, actualAPIKey)
			}

			// verify  the canonical path is correctly bound by checking Viper directly
			viperValue := manager.Viper().GetString(tt.canonicalPath)
			if viperValue != tt.envValue {
				t.Errorf("Expected Viper canonical path '%s' to have value '%s', got '%s'",
					tt.canonicalPath, tt.envValue, viperValue)
			}
		})
	}
}

// TestConfigAliases verifies that aliases work correctly for API key management
func TestConfigAliases(t *testing.T) {
	tests := []struct {
		name          string
		alias         string
		canonicalPath string
		envVar        string
		envValue      string
	}{
		{
			name:          "Mistral alias",
			alias:         "mistral-key",
			canonicalPath: "providers.mistral.api_key",
			envVar:        "MISTRAL_API_KEY",
			envValue:      "test-mistral-alias",
		},
		{
			name:          "Cohere alias",
			alias:         "cohere-key",
			canonicalPath: "providers.cohere.api_key",
			envVar:        "COHERE_API_KEY",
			envValue:      "test-cohere-alias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clean environment
			os.Unsetenv(tt.envVar)

			// set environment variable
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			// create manager and load config
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.toml")

			manager := NewManager()
			err := manager.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// verify alias points to the same value as canonical path
			aliasValue := manager.Viper().GetString(tt.alias)
			canonicalValue := manager.Viper().GetString(tt.canonicalPath)

			if aliasValue != tt.envValue {
				t.Errorf("Expected alias '%s' to have value '%s', got '%s'",
					tt.alias, tt.envValue, aliasValue)
			}

			if canonicalValue != tt.envValue {
				t.Errorf("Expected canonical path '%s' to have value '%s', got '%s'",
					tt.canonicalPath, tt.envValue, canonicalValue)
			}

			if aliasValue != canonicalValue {
				t.Errorf("Alias '%s' and canonical path '%s' should have the same value. Alias: '%s', Canonical: '%s'",
					tt.alias, tt.canonicalPath, aliasValue, canonicalValue)
			}
		})
	}
}

// TestEnvVarOverridesConfig verifies that env var override config file values
// note that we're not doing table-driven tests here – just single case for override
func TestEnvVarOverridesConfig(t *testing.T) {
	// set environment
	os.Unsetenv("COHERE_API_KEY")
	os.Setenv("COHERE_API_KEY", "env-cohere-key")
	defer os.Unsetenv("COHERE_API_KEY")

	// create config file with API key
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")
	configContent := `[providers.cohere]
api_key = "config-cohere-key"`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// load config via NewManager
	manager := NewManager()
	err = manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// verify env var overrides config file
	if manager.cfg.Providers.Cohere.APIKey != "env-cohere-key" {
		t.Errorf("Expected env var to override config: got %q, want %q",
			manager.cfg.Providers.Cohere.APIKey, "env-cohere-key")
	}
}

// TestPostProcessConfig_SeedZeroToNil verifies that seed of 0 gets converted to nil
func TestPostProcessConfig_SeedZeroToNil(t *testing.T) {
	tests := []struct {
		name         string
		seedValue    string
		expectedSeed *int
	}{
		{
			name:         "seed=0 converts to nil",
			seedValue:    "0",
			expectedSeed: nil,
		},
		{
			name:         "seed=42 preserves value",
			seedValue:    "42",
			expectedSeed: &[]int{42}[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create config file with seed value
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.toml")
			configContent := fmt.Sprintf(`[parameters]
seed = %s
temperature = 0.8`, tt.seedValue)

			err := os.WriteFile(configPath, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// load config
			manager := NewManager()
			err = manager.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// verify seed handling
			if tt.expectedSeed == nil {
				if manager.cfg.Parameters.Seed != nil {
					t.Errorf("Expected seed to be nil, got %v", *manager.cfg.Parameters.Seed)
				}
			} else {
				if manager.cfg.Parameters.Seed == nil {
					t.Errorf("Expected seed to be %v, got nil", *tt.expectedSeed)
				} else if *manager.cfg.Parameters.Seed != *tt.expectedSeed {
					t.Errorf("Expected seed to be %v, got %v", *tt.expectedSeed, *manager.cfg.Parameters.Seed)
				}
			}

			// verify other config values are preserved
			if manager.cfg.Parameters.Temperature != 0.8 {
				t.Errorf("Expected temperature to be 0.8, got %f", manager.cfg.Parameters.Temperature)
			}
		})
	}
}
