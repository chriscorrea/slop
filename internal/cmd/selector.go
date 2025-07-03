package cmd

import (
	"fmt"
	"strings"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/spf13/cobra"
)

// ModelSelector handles model selection based on flags and command hints
type ModelSelector interface {
	SelectModel(cmd *cobra.Command, cfg *config.Config, originalArgs []string) (providerName, modelName string, err error)
}

// DefaultModelSelector implements ModelSelector
type DefaultModelSelector struct{}

// NewModelSelector creates a new DefaultModelSelector
func NewModelSelector() *DefaultModelSelector {
	return &DefaultModelSelector{}
}

// SelectModel determines which provider and model to use based on CLI flags / command hints
// TODO: this doesn't need to be so tightly coupled to the *cobra.Command type
func (s *DefaultModelSelector) SelectModel(cmd *cobra.Command, cfg *config.Config, originalArgs []string) (providerName, modelName string, err error) {
	// apply command hints if CLI flags are not explicitly set
	if err := s.applyCommandHints(cmd, cfg, originalArgs); err != nil {
		return "", "", err
	}

	// check for test flag first
	if testFlag, _ := cmd.Flags().GetBool("test"); testFlag {
		return "mock", "test-model", nil
	}

	// determine location pref
	useLocal := false
	if localFlag, _ := cmd.Flags().GetBool("local"); localFlag {
		useLocal = true
	} else if remoteFlag, _ := cmd.Flags().GetBool("remote"); remoteFlag {
		useLocal = false
	}

	// determine deep/fast preference
	useDeep := false
	if deepFlag, _ := cmd.Flags().GetBool("deep"); deepFlag {
		useDeep = true
	} else if fastFlag, _ := cmd.Flags().GetBool("fast"); fastFlag {
		useDeep = false
	}

	// select appropriate model config and then validate it
	var selectedProvider, selectedModel string
	var modelPath string

	if useLocal {
		if useDeep {
			selectedProvider = cfg.Models.Local.Deep.Provider
			selectedModel = cfg.Models.Local.Deep.Name
			modelPath = "models.local.deep"
		} else {
			selectedProvider = cfg.Models.Local.Fast.Provider
			selectedModel = cfg.Models.Local.Fast.Name
			modelPath = "models.local.fast"
		}
	} else {
		if useDeep {
			selectedProvider = cfg.Models.Remote.Deep.Provider
			selectedModel = cfg.Models.Remote.Deep.Name
			modelPath = "models.remote.deep"
		} else {
			selectedProvider = cfg.Models.Remote.Fast.Provider
			selectedModel = cfg.Models.Remote.Fast.Name
			modelPath = "models.remote.fast"
		}
	}

	// validiate the selected model config
	if selectedProvider == "" || selectedModel == "" {
		return "", "", fmt.Errorf("failed to select model: %w", s.generateModelConfigError(modelPath, useLocal, useDeep))
	}

	return selectedProvider, selectedModel, nil
}

// applyCommandHints applies command hints to flags not explicitly set
func (s *DefaultModelSelector) applyCommandHints(cmd *cobra.Command, cfg *config.Config, originalArgs []string) error {
	// only apply hints if we have a named command
	if len(originalArgs) == 0 {
		return nil
	}
	command, exists := cfg.Commands[originalArgs[0]]
	if !exists || command.ModelType == "" {
		return nil
	}

	preset := command.ModelType

	// set --local or --remote flag based on the preset
	if strings.Contains(preset, "local") && !cmd.Flags().Changed("local") {
		if err := cmd.Flags().Set("local", "true"); err != nil {
			return err
		}
	} else if strings.Contains(preset, "remote") && !cmd.Flags().Changed("remote") {
		if err := cmd.Flags().Set("remote", "true"); err != nil {
			return err
		}
	}

	// set --deep or --fast flag based on the preset
	if strings.Contains(preset, "deep") && !cmd.Flags().Changed("deep") {
		if err := cmd.Flags().Set("deep", "true"); err != nil {
			return err
		}
	} else if strings.Contains(preset, "fast") && !cmd.Flags().Changed("fast") {
		if err := cmd.Flags().Set("fast", "true"); err != nil {
			return err
		}
	}
	return nil
}

// generateModelConfigError for helpful error message when model config is missing
func (s *DefaultModelSelector) generateModelConfigError(modelPath string, useLocal, useHeavy bool) error {
	location := "remote"
	if useLocal {
		location = "local"
	}

	modelType := "fast"
	if useHeavy {
		modelType = "deep"
	}

	return fmt.Errorf(`Model configuration missing for %s %s model.

Configure a model with:
  slop config set %s.provider=PROVIDER_NAME
  slop config set %s.name=MODEL_NAME

Example:
  slop config set %s.provider=Mistral %s.name=mistral-medium-latest
a`,
		location, modelType, modelPath, modelPath, modelPath, modelPath)
}
