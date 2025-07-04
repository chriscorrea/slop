package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/chriscorrea/slop/internal/app"
	"github.com/chriscorrea/slop/internal/config"
	slopContext "github.com/chriscorrea/slop/internal/context"
	"github.com/chriscorrea/slop/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// current version (hardcoded for now, could be replaced with build flags)
const version = "0.1.0"

// rootCmdState holds the config manager and logger for the command
type rootCmdState struct {
	manager *config.Manager
	logger  *slog.Logger
}

// state is the global state instance for the root command
var state = &rootCmdState{}

// expandHomePath expands ~ to the user's home directory
func expandHomePath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if len(path) == 1 {
		return home, nil
	}

	return filepath.Join(home, path[1:]), nil
}

// showCustomHelp is structured for Cobra subcommands and named commands
func showCustomHelp(cmd *cobra.Command) {
	// show the standard help first
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", cmd.Long)

	fmt.Fprintf(cmd.OutOrStdout(), "Usage:\n  %s [flags]\n  %s [command] [args...]\n\n", cmd.Use, cmd.Use)

	// show Cobra subcommands and reserved commands
	fmt.Fprintln(cmd.OutOrStdout(), "Available Commands:")
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", subCmd.Name(), subCmd.Short)
		}
	}

	// show named commands (if config is available)
	if state.manager != nil {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Named Commands:")
		baseConfig := state.manager.Config()
		for cmdName, command := range baseConfig.Commands {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", cmdName, command.Description)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Flags:")
	// print flags manually to get proper output formatting
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flagStr := fmt.Sprintf("      --%s", flag.Name)
		if flag.Shorthand != "" {
			flagStr = fmt.Sprintf("  -%s, --%s", flag.Shorthand, flag.Name)
		}

		// add type information for non-boolflags
		if flag.Value.Type() != "bool" {
			flagStr += fmt.Sprintf(" %s", flag.Value.Type())
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%-30s %s", flagStr, flag.Usage)
		if flag.DefValue != "" && flag.DefValue != "false" && flag.DefValue != "[]" {
			fmt.Fprintf(cmd.OutOrStdout(), " (default %s)", flag.DefValue)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	})

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Use \"slop [command] --help\" for more information about core commands such as list, config, and init")
	fmt.Fprintln(cmd.OutOrStdout(), "Use \"slop help-command [your-command]\" for help on a user-defined, custom named command")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "slop",
	Version:      version,
	Short:        "A CLI tool for interacting with LLMs",
	Long:         `Slop brings large language models to your command line. It is inspired by the idea that language models work best as composable language operators`,
	SilenceUsage: true, // Don't show usage after errors

	// accept any arguments and pass them to RunE
	Args: cobra.ArbitraryArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// get the debug flag value and create logger
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("failed to get debug flag: %w", err)
		}
		state.logger = logger.New(debug)

		// instantiate the config manager with logger
		state.manager = config.NewManager().WithLogger(state.logger)

		// Get the config flag value
		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("failed to get config flag: %w", err)
		}

		// if config is not set, use default path
		if configPath == "" {
			configPath = "~/.slop/config.toml"
		}

		// expand home directory if needed
		configPath, err = expandHomePath(configPath)
		if err != nil {
			return fmt.Errorf("failed to expand home path: %w", err)
		}

		// bind all persistent flags to their corresponding Viper keys
		viper := state.manager.Viper()

		// binding map
		flagBindings := map[string]string{
			"system":         "parameters.system_prompt",
			"context":        "context",
			"ignore-context": "no_context",
			"local":          "local",
			"fast":           "fast",
			"deep":           "deep",
			"temperature":    "parameters.temperature",
			"json":           "format.json",
			"jsonl":          "format.jsonl",
			"yaml":           "format.yaml",
			"md":             "format.md",
			"xml":            "format.xml",
			"verbose":        "verbose",
			"debug":          "debug",
			"seed":           "parameters.seed",
			"max-tokens":     "parameters.max_tokens",
			"max-retries":    "parameters.max_retries",
			"timeout":        "parameters.timeout",
			"test":           "test",
		}

		// bind each flag to corresponding Viper key
		for flagName, viperKey := range flagBindings {
			if err := viper.BindPFlag(viperKey, cmd.Flags().Lookup(flagName)); err != nil {
				return fmt.Errorf("failed to bind flag %s: %w", flagName, err)
			}
		}

		// load the config
		err = state.manager.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// if no arguments provided, show custom help
		if len(args) == 0 {
			showCustomHelp(cmd)
			return nil
		}

		// check if first argument is a named command
		if state.manager != nil {
			cfg := state.manager.Config()
			if cmdConfig, exists := cfg.Commands[args[0]]; exists && !config.ReservedCommands[args[0]] {
				// Handle as named command
				return handleNamedCommand(cmd, args[0], cmdConfig, args[1:])
			}
		}

		// handle direct prompts (no named command)
		return handleDirectPrompt(cmd, args)
	},
}

// execute adds all child commands to the root command and sets flags
// this is called by main.main() – it only needs to happen once to the rootCmd
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// define persistent flags
	rootCmd.PersistentFlags().String("config", "", "Path to the config file")
	rootCmd.PersistentFlags().String("system", "", "The system prompt")
	rootCmd.PersistentFlags().StringSlice("context", []string{}, "Path to context file(s)")
	rootCmd.PersistentFlags().BoolP("ignore-context", "i", false, "Ignore project context for this command")
	rootCmd.PersistentFlags().BoolP("local", "l", false, "Use local LLM")
	rootCmd.PersistentFlags().BoolP("remote", "r", false, "Use remote LLM")
	rootCmd.PersistentFlags().BoolP("fast", "f", false, "Use fast/lightweight model")
	rootCmd.PersistentFlags().BoolP("deep", "d", false, "Use deep/reasoning model")
	rootCmd.PersistentFlags().Bool("test", false, "Use mock provider for testing")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Display LLM parameters in formatted table")
	rootCmd.PersistentFlags().BoolP("debug", "D", false, "Enable detailed debug logging")

	rootCmd.PersistentFlags().Float64("temperature", 0.7, "Temperature for LLM responses")
	rootCmd.PersistentFlags().Int("max-tokens", 2048, "Maximum number of tokens for LLM responses")
	rootCmd.PersistentFlags().Float64("top-p", 1.0, "Top P sampling for LLM responses")
	rootCmd.PersistentFlags().StringSlice("stop-sequences", []string{"\n", "###"}, "Stop sequences for LLM responses")
	rootCmd.PersistentFlags().Bool("stream", true, "Enable streaming responses from LLM") // TODO
	rootCmd.PersistentFlags().Int("seed", 0, "Random seed for deterministic LLM outputs (0 = no seed)")

	rootCmd.PersistentFlags().Int("timeout", 60, "Timeout in seconds for LLM requests")
	rootCmd.PersistentFlags().Int("max-retries", 1, "Maximum number of retry attempts for failed requests (max: 5)")

	// Output formatting flags
	rootCmd.PersistentFlags().Bool("json", false, "Format response as JSON")
	rootCmd.PersistentFlags().Bool("jsonl", false, "Format response as JSONL (JSON Lines)")
	rootCmd.PersistentFlags().Bool("yaml", false, "Format response as YAML")
	rootCmd.PersistentFlags().Bool("md", false, "Format response as Markdown")
	rootCmd.PersistentFlags().Bool("xml", false, "Format response as XML")

	// mark the mutually exclusive flags
	rootCmd.MarkFlagsMutuallyExclusive("fast", "deep")
	rootCmd.MarkFlagsMutuallyExclusive("local", "remote")
	rootCmd.MarkFlagsMutuallyExclusive("json", "jsonl", "yaml", "md", "xml")

	// list of flags to hide for now
	flagsToHide := []string{"test", "stream"}

	for _, flagName := range flagsToHide {
		err := rootCmd.PersistentFlags().MarkHidden(flagName)
		if err != nil {
			// this shouldn't happen in production, but it's good practice to catch? ¯\_(ツ)_/¯
			panic(err)
		}
	}

	// custom usage template will hide lengthly global flags list for subcommands
	rootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	// add reserved commands as proper Cobra subcommands
	rootCmd.AddCommand(createListCommand())
	rootCmd.AddCommand(createVersionCommand())
	rootCmd.AddCommand(createNamedHelpCommand())
	rootCmd.AddCommand(createContextCommand())
}

// executeApp handles the common execution logic for both direct prompts and named commands
func executeApp(cmd *cobra.Command, args []string, cfg *config.Config, contextResult *slopContext.ContextResult, commandContext string, showCommandInfo bool, commandName string) error {
	// select model using the selector
	providerName, modelName, err := selectModelForCommand(cmd, cfg, commandName, args)
	if err != nil {
		return fmt.Errorf("failed to select model: %w", err)
	}

	// get verbose flag to determine output mode
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return fmt.Errorf("failed to get verbose flag: %w", err)
	}

	// show command usage information if requested
	if showCommandInfo {
		if cmdConfig, exists := cfg.Commands[commandName]; exists {
			fmt.Fprintf(cmd.ErrOrStderr(), "Using command: %s - %s\n", commandName, cmdConfig.Description)
		}
	}

	// create app with config, logger, and verbose setting
	appInstance := app.NewApp(cfg, state.logger, verbose)

	// run the app
	output, err := appInstance.Run(
		cmd.Context(),
		args, // user prompt arguments
		contextResult,
		commandContext, // command context (empty for a direct prompt)
		providerName,
		modelName,
	)
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}

// handleDirectPrompt handles direct prompts when no named command is used
func handleDirectPrompt(cmd *cobra.Command, args []string) error {
	if state.manager == nil {
		return fmt.Errorf("config manager not initialized")
	}

	// use base config without any command overrides
	cfg := state.manager.Config()

	// check for --ignore-context flag
	skipProjectContext, err := cmd.Flags().GetBool("ignore-context")
	if err != nil {
		return fmt.Errorf("failed to get ignore-context flag: %w", err)
	}

	// process context using the context manager
	contextManager := NewContextManager()
	contextResult, err := contextManager.ProcessContextWithFlags(cmd, nil, skipProjectContext)
	if err != nil {
		return fmt.Errorf("failed to process context: %w", err)
	}

	// exec app with no command context, no command info display
	return executeApp(cmd, args, cfg, contextResult, "", false, "")
}

// selectModelForCommand uses the existing model selector logic
func selectModelForCommand(cmd *cobra.Command, cfg *config.Config, cmdName string, args []string) (string, string, error) {

	// Create a model selector and use it
	selector := NewModelSelector()

	providerName, modelName, err := selector.SelectModel(cmd, cfg, args)
	if err != nil {
		return "", "", err
	}

	// log model selection
	if state.logger != nil {
		if cmdName != "" {
			state.logger.Info("Model selected for named command", "command", cmdName, "model_name", modelName, "provider", providerName)
		} else {
			state.logger.Info("Model selected for direct prompt", "model_name", modelName, "provider", providerName)
		}
	}

	return providerName, modelName, nil
}

// handleNamedCommand handles execution of a named command
func handleNamedCommand(cmd *cobra.Command, cmdName string, cmdConfig config.Command, args []string) error {
	if state.manager == nil {
		return fmt.Errorf("config manager not initialized")
	}

	// apply command overrides to get working config
	baseConfig := state.manager.Config()
	workingConfig := baseConfig.WithCommandOverrides(cmdConfig)

	// check for --ignore-context flag
	skipProjectContext, err := cmd.Flags().GetBool("ignore-context")
	if err != nil {
		return fmt.Errorf("failed to get ignore-context flag: %w", err)
	}

	// process context using the context manager with command context files
	contextManager := NewContextManager()
	contextResult, err := contextManager.ProcessContextWithFlags(cmd, cmdConfig.ContextFiles, skipProjectContext)
	if err != nil {
		return fmt.Errorf("failed to process context: %w", err)
	}

	// exec app with command context and command info display
	return executeApp(cmd, args, workingConfig, contextResult, cmdConfig.Context, true, cmdName)
}

// createListCommand creates the list subcommand
func createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available commands",
		Long:  "Display all available commands including both built-in and custom named commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.manager == nil {
				return fmt.Errorf("config manager not initialized")
			}

			baseConfig := state.manager.Config()

			fmt.Fprintln(cmd.OutOrStdout(), "Available commands:")
			fmt.Fprintln(cmd.OutOrStdout())

			// show built-in commands
			fmt.Fprintln(cmd.OutOrStdout(), "Built-in commands:")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", "help", "Show help for commands")
			fmt.Fprintln(cmd.OutOrStdout())

			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", "config", "Show configuration information")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", "config set", "Set a specific configuration value")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", "context", "Manage persistent context for the current directory")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", "list", "List all available commands")

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", "init", "Configure a new slop installation")

			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Named commands:")

			// show custom commands
			for cmdName, command := range baseConfig.Commands {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s\n", cmdName, command.Description)
			}

			return nil
		},
	}
}

// createVersionCommand creates the version subcommand
func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the current version of slop.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), "slop version ", version, "\n")
			return nil
		},
	}
}

// createNamedHelpCommand creates a help command that can show details for named commands
func createNamedHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help-command [command]",
		Short: "Show help for user-defined, named commands",
		Long:  "Display detailed information about named commands defined in your configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Usage: slop help-command <command-name>")
				fmt.Fprintln(cmd.OutOrStdout(), "")
				fmt.Fprintln(cmd.OutOrStdout(), "Use 'slop list' to see all available named commands.")
				return nil
			}

			if state.manager == nil {
				return fmt.Errorf("config manager not initialized")
			}

			// provide help for specific user-defined, named command
			commandName := args[0]
			baseConfig := state.manager.Config()
			if command, exists := baseConfig.Commands[commandName]; exists {
				fmt.Fprintf(cmd.OutOrStdout(), "Command: %s\n", commandName)
				fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", command.Description)

				if command.SystemPrompt != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "System Prompt: %s\n", command.SystemPrompt)
				}

				// show command settings
				var settings []string
				if command.ModelType != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Model Type: %s\n", command.ModelType)
				}
				if command.Temperature != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Temperature: %.1f\n", *command.Temperature)
				}
				if command.MaxTokens != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Max Tokens: %d\n", *command.MaxTokens)
				}
				if len(settings) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Settings: %s\n", strings.Join(settings, ", "))
				}

				if len(command.ContextFiles) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Context Files: %s\n", strings.Join(command.ContextFiles, ", "))
				}

				return nil
			}

			return fmt.Errorf("unknown named command: %s", commandName)
		},
	}
}
