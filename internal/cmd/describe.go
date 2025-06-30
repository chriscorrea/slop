package cmd

import (
	"fmt"

	"slop/internal/config"

	"github.com/spf13/cobra"
)

// describeConfigCmd represents the config describe command
var describeConfigCmd = &cobra.Command{
	Use:   "describe <key>",
	Short: "Show detailed information about a configuration key",
	Long: `Show detailed information about a configuration key including its type,
description, and value.

The key can be either a full canonical path or a convenience alias.

Examples:
  slop config describe temperature
  slop config describe parameters.temperature`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		schema := config.DefaultConfigSchema()

		canonicalKey, err := schema.ResolveKey(key)
		if err != nil {
			return err
		}

		// get field info
		fieldInfo, err := schema.GetFieldInfo(canonicalKey)
		if err != nil {
			return err
		}

		// get current value
		// note getConfigValue is in list.go
		currentValue := getConfigValue(canonicalKey)

		// display information
		fmt.Fprintf(cmd.OutOrStdout(), "Configuration Key: %s\n", canonicalKey)

		// show alias if the input was an alias
		if key != canonicalKey {
			fmt.Fprintf(cmd.OutOrStdout(), "Alias: %s\n", key)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", fieldInfo.Type.String())
		fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", fieldInfo.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "Current Value: %v\n", currentValue)

		// show validation info if available
		if fieldInfo.Validation != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nValidation: Custom validation rules apply\n")
		}

		// show related aliases (for  canonical path)
		var relatedAliases []string
		for alias, canonical := range schema.Aliases {
			if canonical == canonicalKey && alias != key {
				relatedAliases = append(relatedAliases, alias)
			}
		}

		if len(relatedAliases) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nAliases: %v\n", relatedAliases)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(describeConfigCmd)
}
