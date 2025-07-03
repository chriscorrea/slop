package cmd

import (
	"fmt"

	"github.com/chriscorrea/slop/internal/data"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize slop config through an interactive process",
	Long: `Initialize your slop configuration:
• Configure remote LLM provider
• Set up local endpoint
• Choose optimal models for different use cases
• Store API key(s)

Your configuration will be saved to ~/.slop/config.toml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// create color functions for consistent styling
		cyan := color.New(color.FgCyan).SprintFunc()
		magenta := color.New(color.FgMagenta).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()

		// welcome message
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s\n", cyan("🐷 Welcome to slop"))
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s\n", "Let's get you set up–this will only take a minute!")

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
		err = survey.AskOne(remotePrompt, &configureRemote)
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
				Default: providerOptions[0], // default to first option
			}
			err = survey.AskOne(providerPrompt, &selectedProvider)
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
			err = survey.AskOne(apiKeyPrompt, &apiKey)
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
				Help:    "This model will be used for quick, everyday tasks",
			}
			err = survey.AskOne(fastPrompt, &fastModel)
			if err != nil {
				return fmt.Errorf("survey error: %w", err)
			}

			// configure deep model
			var deepModel string
			deepPrompt := &survey.Input{
				Message: fmt.Sprintf("%s Deep model for reasoning tasks:", cyan("🧠")),
				Default: providerInfo.Models.Deep,
				Help:    "This model will be used for complex analysis and reasoning tasks",
			}
			err = survey.AskOne(deepPrompt, &deepModel)
			if err != nil {
				return fmt.Errorf("survey error: %w", err)
			}

			// set model configurations
			viper.Set("models.remote.fast.provider", providerKey)
			viper.Set("models.remote.fast.name", fastModel)
			viper.Set("models.remote.deep.provider", providerKey)
			viper.Set("models.remote.deep.name", deepModel)

			fmt.Fprintf(cmd.ErrOrStderr(), "\n%s %s configured successfully!\n", green("✅"), providerInfo.Name)
		}

		// Configure local provider
		var configureLocal bool
		localPrompt := &survey.Confirm{
			Message: fmt.Sprintf("%s Would you like to configure a local AI provider (Ollama)?", cyan("🏠")),
			Default: false,
			Help:    "Ollama runs AI models locally on your machine",
		}
		err = survey.AskOne(localPrompt, &configureLocal)
		if err != nil {
			return fmt.Errorf("survey error: %w", err)
		}

		if configureLocal {
			ollamaInfo, exists := registry.GetProvider("ollama")
			if !exists {
				return fmt.Errorf("ollama provider not found in configuration")
			}

			// Configure Ollama URL
			var ollamaURL string
			urlPrompt := &survey.Input{
				Message: fmt.Sprintf("%s Ollama server URL:", cyan("🔗")),
				Default: "http://127.0.0.1:11434",
				Help:    "The URL where your Ollama server is running",
			}
			err = survey.AskOne(urlPrompt, &ollamaURL)
			if err != nil {
				return fmt.Errorf("survey error: %w", err)
			}

			// configure local fast model
			var localFastModel string
			localFastPrompt := &survey.Input{
				Message: fmt.Sprintf("%s Local fast model:", cyan("⚡")),
				Default: ollamaInfo.Models.Fast,
				Help:    "Local model for quick responses",
			}
			err = survey.AskOne(localFastPrompt, &localFastModel)
			if err != nil {
				return fmt.Errorf("survey error: %w", err)
			}

			// configure local deep model
			var localDeepModel string
			localDeepPrompt := &survey.Input{
				Message: fmt.Sprintf("%s Local deep model:", cyan("🧠")),
				Default: ollamaInfo.Models.Deep,
				Help:    "Local model for complex reasoning",
			}
			err = survey.AskOne(localDeepPrompt, &localDeepModel)
			if err != nil {
				return fmt.Errorf("survey error: %w", err)
			}

			// set local configurations
			viper.Set("providers.ollama.base_url", ollamaURL)
			viper.Set("models.local.fast.provider", "ollama")
			viper.Set("models.local.fast.name", localFastModel)
			viper.Set("models.local.deep.provider", "ollama")
			viper.Set("models.local.deep.name", localDeepModel)

			fmt.Fprintf(cmd.ErrOrStderr(), "\n%s Ollama configured successfully!\n", green("✅"))
		}

		// Save configuration
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s Saving your configuration...\n", yellow("💾"))

		err = viper.SafeWriteConfig()
		if err != nil {
			// if config file already exists, we might need to write anyway
			if err = viper.WriteConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}
		}

		// --- conclusion messages ---
		configPath := viper.ConfigFileUsed()
		if configPath == "" {
			configPath = "~/.slop/config.toml"
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s All set! Your configuration has been saved to %s\n",
			green("🎉"), magenta(configPath))
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s You can now start using slop! Try: %s\n",
			cyan("💡"), magenta("slop \"What is the nature of this life of ours?\""))
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s For more options, run: %s\n\n",
			cyan("📖"), magenta("slop --help"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
