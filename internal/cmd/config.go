package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage slop configuration",
	Long: `Manage slop configuration settings. This command provides subcommands
to view and modify configuration values.

Examples:
  slop config            		# Show current configuration status
  slop config set key=value   	# Set a configuration value

      slop config set cohere_key=<your api key>
      slop config set parameters.temperature=0.7
      slop config set temperature=0.7
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// show configuration information when no subcommand is provided
		fmt.Fprintln(cmd.OutOrStdout(), "Configuration loaded successfully")
		fmt.Fprintf(cmd.OutOrStdout(), "Available commands: %d\n", len(state.manager.Config().Commands))

		// show configuration file location
		if state.manager.Viper().ConfigFileUsed() != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", state.manager.Viper().ConfigFileUsed())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
