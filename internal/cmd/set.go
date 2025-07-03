package cmd

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set <key>=<value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value in the config file.

The key should be in dot notation format (e.g., parameters.temperature),
but convenient aliases are also supported.

For example, you can use convenient aliases for API keys:
  cohere_key     → providers.cohere.api_key
  mistral_key    → providers.mistral.api_key  

Examples:
  slop config set parameters.temperature=0.5772
  slop config set parameters.system_prompt="You are helpful"
  slop config set parameters.max_tokens=1024
  slop config set models.remote.light.provider=mistral
  slop config set anthropic_key=ak-65433210`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse the key=value argument
		argument := args[0]
		parts := strings.SplitN(argument, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: expected key=value, got %q", argument)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return fmt.Errorf("key cannot be empty")
		}

		// Create schema for validation
		schema := config.DefaultConfigSchema()

		// Resolve key (handle aliases)
		canonicalKey, err := schema.ResolveKey(key)
		if err != nil {
			return err
		}

		// Get field info for type conversion
		fieldInfo, err := schema.GetFieldInfo(canonicalKey)
		if err != nil {
			return err
		}

		// Convert value to the expected type
		convertedValue, err := convertValueToType(value, fieldInfo.Type)
		if err != nil {
			return fmt.Errorf("failed to convert value %q for key %q: %w", value, canonicalKey, err)
		}

		// Validate the converted value
		if err := schema.ValidateValue(canonicalKey, convertedValue); err != nil {
			return fmt.Errorf("validation failed for key %q: %w", canonicalKey, err)
		}

		// Get the manager and set the value
		manager := state.manager
		viper := manager.Viper()

		// Set the value in Viper using the canonical key
		viper.Set(canonicalKey, convertedValue)

		// Save the configuration
		if err := manager.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		// Print confirmation showing both alias and canonical key if different
		if key != canonicalKey {
			fmt.Fprintf(cmd.OutOrStdout(), "Configuration updated: %s (%s) = %v\n", key, canonicalKey, convertedValue)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Configuration updated: %s = %v\n", canonicalKey, convertedValue)
		}

		return nil
	},
}

// convertValueToType converts a string value to the specified type
func convertValueToType(value string, targetType reflect.Type) (interface{}, error) {
	// Try to unquote the value if it appears to be quoted
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'' || value[0] == '`') {
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		// If unquoting fails, use the original value (could be a bare string that starts with a quote)
	}

	switch targetType.Kind() {
	case reflect.String:
		return value, nil

	case reflect.Bool:
		return strconv.ParseBool(strings.ToLower(value))

	case reflect.Int:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(intVal), nil

	case reflect.Int64:
		return strconv.ParseInt(value, 10, 64)

	case reflect.Float64:
		return strconv.ParseFloat(value, 64)

	case reflect.Float32:
		floatVal, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return nil, err
		}
		return float32(floatVal), nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", targetType.String())
	}
}

func init() {
	configCmd.AddCommand(setCmd)
}
