package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/data"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

// setupProviderData creates minimal provider data for testing
func setupProviderData(t *testing.T) func() {
	// create temp directory for configs
	tempDir, err := os.MkdirTemp("", "slop-provider-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for provider data: %v", err)
	}

	configsDir := filepath.Join(tempDir, "configs")
	err = os.MkdirAll(configsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create configs dir: %v", err)
	}

	// create minimal models.json
	modelsJSON := `{
		"providers": {
			"entropic": {
				"name": "Entropic",
				"description": "Models from Entropic",
				"models": {
					"fast": "entropic-sestina-latest",
					"deep": "entropic-villanelle-4-0"
				}
			}
		}
	}`

	modelsPath := filepath.Join(configsDir, "models.json")
	err = os.WriteFile(modelsPath, []byte(modelsJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test models.json: %v", err)
	}

	// change to temp directory so provider registry finds our test file
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// return cleanup function
	return func() {
		_ = os.Chdir(originalDir)
		os.RemoveAll(tempDir)
	}
}

// runSimpleInit runs the init logic with mocked survey responses
func runSimpleInit(mockAskOne func(survey.Prompt, interface{}) error) error {
	// create color functions for consistent styling
	cyan := color.New(color.FgCyan).SprintFunc()

	// initialize provider registry
	registry := data.NewProviderRegistry()
	err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load provider data: %w", err)
	}

	// get Viper instance
	viper := state.manager.Viper()

	// configure remote provider
	var configureRemote bool
	remotePrompt := &survey.Confirm{
		Message: fmt.Sprintf("%s Would you like to configure a remote AI provider?", cyan("🌐")),
		Default: true,
	}
	err = mockAskOne(remotePrompt, &configureRemote)
	if err != nil {
		return fmt.Errorf("survey error: %w", err)
	}

	if configureRemote {
		// get provider options
		providerOptions := registry.GetProviderOptions()

		var selectedProvider string
		providerPrompt := &survey.Select{
			Message: fmt.Sprintf("%s Choose your preferred remote AI provider:", cyan("🤖")),
			Options: providerOptions,
			Default: providerOptions[0],
		}
		err = mockAskOne(providerPrompt, &selectedProvider)
		if err != nil {
			return fmt.Errorf("survey error: %w", err)
		}

		providerKey := registry.GetProviderKeyFromOption(selectedProvider)
		if providerKey == "" {
			return fmt.Errorf("failed to determine provider key")
		}

		providerInfo, exists := registry.GetProvider(providerKey)
		if !exists {
			return fmt.Errorf("provider %s not found", providerKey)
		}

		// configure API key for remote provider
		var apiKey string
		apiKeyPrompt := &survey.Password{
			Message: fmt.Sprintf("%s Enter your %s API key:", cyan("🔑"), providerInfo.Name),
		}
		err = mockAskOne(apiKeyPrompt, &apiKey)
		if err != nil {
			return fmt.Errorf("survey error: %w", err)
		}

		// set API key in config
		viper.Set(fmt.Sprintf("providers.%s.api_key", providerKey), apiKey)

		// configure fast model
		var fastModel string
		fastPrompt := &survey.Input{
			Message: fmt.Sprintf("%s Default fast model for everyday tasks:", cyan("⚡")),
			Default: providerInfo.Models.Fast,
		}
		err = mockAskOne(fastPrompt, &fastModel)
		if err != nil {
			return fmt.Errorf("survey error: %w", err)
		}

		// configure deep model
		var deepModel string
		deepPrompt := &survey.Input{
			Message: fmt.Sprintf("%s Deep model for reasoning tasks:", cyan("🧠")),
			Default: providerInfo.Models.Deep,
		}
		err = mockAskOne(deepPrompt, &deepModel)
		if err != nil {
			return fmt.Errorf("survey error: %w", err)
		}

		// set model configurations
		viper.Set("models.remote.fast.provider", providerKey)
		viper.Set("models.remote.fast.name", fastModel)
		viper.Set("models.remote.deep.provider", providerKey)
		viper.Set("models.remote.deep.name", deepModel)
	}

	// save configuration
	err = viper.SafeWriteConfig()
	if err != nil {
		if err = viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	}

	return nil
}

// TestInitCommand tests the basic functionality of the init command
func TestInitCommand(t *testing.T) {
	// create temporary directory for config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// setup config manager with temp directory
	manager := config.NewManager()
	err := manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config manager: %v", err)
	}

	// setup global state for test
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// setup provider data
	providerCleanup := setupProviderData(t)
	defer providerCleanup()

	// simple mock function
	mockAskOne := func(prompt survey.Prompt, response interface{}) error {
		switch p := prompt.(type) {
		case *survey.Confirm:
			if strings.Contains(p.Message, "remote AI provider") {
				*response.(*bool) = true
			} else if strings.Contains(p.Message, "local AI provider") {
				*response.(*bool) = false
			}
		case *survey.Select:
			*response.(*string) = "Anthropic - Claude models from Anthropic"
		case *survey.Password:
			*response.(*string) = "test-api-key-12345"
		case *survey.Input:
			if strings.Contains(p.Message, "fast model") {
				*response.(*string) = "claude-3-5-haiku-latest"
			} else if strings.Contains(p.Message, "Deep model") {
				*response.(*string) = "claude-sonnet-4-0"
			}
		}
		return nil
	}

	// run the simplified init logic
	err = runSimpleInit(mockAskOne)
	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	// verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// verify some key configuration values were set
	viper := manager.Viper()

	// check that API key was set
	apiKey := viper.GetString("providers.anthropic.api_key")
	if apiKey != "test-api-key-12345" {
		t.Errorf("Expected API key to be 'test-api-key-12345', got '%s'", apiKey)
	}

	// check that models were configured
	fastProvider := viper.GetString("models.remote.fast.provider")
	if fastProvider != "anthropic" {
		t.Errorf("Expected fast model provider to be 'anthropic', got '%s'", fastProvider)
	}

	fastModel := viper.GetString("models.remote.fast.name")
	if fastModel != "claude-3-5-haiku-latest" {
		t.Errorf("Expected fast model to be 'claude-3-5-haiku-latest', got '%s'", fastModel)
	}
}

// TestInitCommand_SurveyError tests error handling when survey fails
func TestInitCommand_SurveyError(t *testing.T) {
	// Create temporary directory for config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Setup config manager
	manager := config.NewManager()
	err := manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config manager: %v", err)
	}

	// setup global state for test
	originalState := state
	state = &rootCmdState{manager: manager}
	defer func() { state = originalState }()

	// setup provider data
	providerCleanup := setupProviderData(t)
	defer providerCleanup()

	// mock survey function that returns an error
	mockAskOne := func(prompt survey.Prompt, response interface{}) error {
		return fmt.Errorf("survey failed")
	}

	// run the init command and expect error
	err = runSimpleInit(mockAskOne)
	if err == nil {
		t.Fatal("Expected init command to fail when survey returns error")
	}

	expectedError := "survey error"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}
