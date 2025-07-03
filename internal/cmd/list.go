package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ConfigDisplayInfo holds information for displaying config item
type ConfigDisplayInfo struct {
	Key         string
	Value       string
	Description string
	Group       string
	IsAlias     bool
	Target      string // the canonical path an alias points to
}

// OutputStyle contains color configuration for the list output
type OutputStyle struct {
	Writer       io.Writer
	KeyColor     *color.Color
	ValueColor   *color.Color
	HeaderColor  *color.Color
	GroupColor   *color.Color
	AliasColor   *color.Color
	EnableColors bool
}

// NewOutputStyle creates new output style configuration
func NewOutputStyle(writer io.Writer) *OutputStyle {
	return &OutputStyle{
		Writer:       writer,
		KeyColor:     color.New(color.FgCyan, color.Bold),
		ValueColor:   color.New(color.FgMagenta),
		HeaderColor:  color.New(color.FgYellow, color.Bold, color.Underline),
		GroupColor:   color.New(color.FgGreen, color.Bold),
		AliasColor:   color.New(color.FgBlue),
		EnableColors: true,
	}
}

// listConfigCmd represents the config list cmd
var listConfigCmd = &cobra.Command{
	Use:   "list",
	Short: "List configuration values",
	Long: `List configuration values.

By default, shows the user-friendly aliases view. Use --canonical to see 
the complete configuration structure with all canonical paths.

Views:
  Default/Aliases: Shows convenient aliases organized by category
  Canonical:       Shows complete configuration structure

Examples:
  slop config list              # Show aliases view (default)
  slop config list --aliases    # Show aliases view (explicit)
  slop config list --canonical  # Show canonical configuration paths`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schema := config.DefaultConfigSchema()
		manager := state.manager
		cfg := manager.Config()

		// get flags
		showCanonical, _ := cmd.Flags().GetBool("canonical")
		showAliases, _ := cmd.Flags().GetBool("aliases")

		// show aliases by default; this is just more human-friendly
		if !showCanonical && !showAliases {
			showAliases = true
		}

		style := NewOutputStyle(cmd.OutOrStdout())

		if showAliases {
			return displayAliasesView(cfg, schema, style)
		} else {
			return displayCanonicalView(cfg, schema, style)
		}
	},
}

// displayAliasesView shows the user-friendly aliases organized by category
func displayAliasesView(cfg *config.Config, schema *config.ConfigSchema, style *OutputStyle) error {
	w := tabwriter.NewWriter(style.Writer, 0, 0, 3, ' ', 0)

	// group aliases by category
	groups := map[string][]ConfigDisplayInfo{
		"Parameters": {},
		"Models":     {},
		"Providers":  {},
		"Format":     {},
	}

	// collect and categorize aliases
	aliases := schema.ListAliases()
	sort.Strings(aliases)

	for _, alias := range aliases {
		canonicalPath := schema.Aliases[alias]
		fieldInfo, err := schema.GetFieldInfo(canonicalPath)
		if err != nil {
			// skip if no field info available
			continue
		}

		value := getConfigValue(canonicalPath)
		maskedValue := maskSensitiveValue(alias, value)

		info := ConfigDisplayInfo{
			Key:         alias,
			Value:       maskedValue,
			Description: fieldInfo.Description,
			IsAlias:     true,
			Target:      canonicalPath,
		}

		// categorize by prefix or content
		// TODO: move local-deep-provider, remote-fast-provider, etc to Models or Providers?
		switch {
		case strings.HasPrefix(alias, "temp") ||
			strings.Contains(alias, "tokens") ||
			strings.Contains(alias, "retries") ||
			strings.Contains(alias, "timeout") ||
			strings.Contains(alias, "system") ||
			strings.Contains(alias, "seed") ||
			strings.Contains(alias, "top-p"):
			groups["Parameters"] = append(groups["Parameters"], info)
		case strings.Contains(alias, "model") ||
			strings.Contains(alias, "location"):
			groups["Models"] = append(groups["Models"], info)
		case strings.Contains(alias, "key") ||
			strings.Contains(alias, "anthropic") ||
			strings.Contains(alias, "openai") ||
			strings.Contains(alias, "mistral") ||
			strings.Contains(alias, "cohere"):
			groups["Providers"] = append(groups["Providers"], info)
		case alias == "json" || alias == "yaml" || alias == "md" || alias == "xml":
			groups["Format"] = append(groups["Format"], info)
		default:
			groups["Parameters"] = append(groups["Parameters"], info)
		}
	}

	// display each group
	groupOrder := []string{"Parameters", "Models", "Providers", "Format"}
	for _, groupName := range groupOrder {
		items := groups[groupName]
		if len(items) == 0 {
			continue
		}

		printSectionHeader(w, style, groupName)
		for _, item := range items {
			printConfigRow(w, style, item.Key, item.Value, item.Description)
		}
		fmt.Fprintf(w, "\n")
	}

	w.Flush()
	return nil
}

// displayCanonicalView shows the complete config structure
func displayCanonicalView(cfg *config.Config, schema *config.ConfigSchema, style *OutputStyle) error {
	w := tabwriter.NewWriter(style.Writer, 0, 0, 3, ' ', 0)

	// get all canonical keys, group them
	canonicalKeys := schema.ListCanonicalKeys()
	sort.Strings(canonicalKeys)

	groups := map[string][]ConfigDisplayInfo{
		"Parameters": {},
		"Models":     {},
		"Providers":  {},
		"Format":     {},
	}

	for _, key := range canonicalKeys {
		fieldInfo, err := schema.GetFieldInfo(key)
		if err != nil {
			continue
		}

		value := getConfigValue(key)
		maskedValue := maskSensitiveValueCanonical(key, value)

		info := ConfigDisplayInfo{
			Key:         key,
			Value:       maskedValue,
			Description: fieldInfo.Description,
			IsAlias:     false,
		}

		// group by top-level key
		switch {
		case strings.HasPrefix(key, "parameters."):
			groups["Parameters"] = append(groups["Parameters"], info)
		case strings.HasPrefix(key, "models."):
			groups["Models"] = append(groups["Models"], info)
		case strings.HasPrefix(key, "providers."):
			groups["Providers"] = append(groups["Providers"], info)
		case strings.HasPrefix(key, "format."):
			groups["Format"] = append(groups["Format"], info)
		default:
			groups["Parameters"] = append(groups["Parameters"], info)
		}
	}

	// display each group
	groupOrder := []string{"Parameters", "Models", "Providers", "Format"}
	for _, groupName := range groupOrder {
		items := groups[groupName]
		if len(items) == 0 {
			continue
		}

		printCanonicalSectionHeader(w, style, groupName)
		for _, item := range items {
			printCanonicalConfigRow(w, style, item.Key, item.Value)
		}
		fmt.Fprintf(w, "\n")
	}

	w.Flush()
	return nil
}

// printSectionHeader prints a section header for grouped config items
func printSectionHeader(w io.Writer, style *OutputStyle, groupName string) {
	groupSprint := style.GroupColor.SprintFunc()
	keySprint := style.KeyColor.SprintFunc()
	valueSprint := style.ValueColor.SprintFunc()

	if !style.EnableColors {
		groupSprint = fmt.Sprint
		keySprint = fmt.Sprint
		valueSprint = fmt.Sprint
	}

	fmt.Fprintf(w, "%s\n", groupSprint(fmt.Sprintf("▶ %s", groupName)))

	fmt.Fprintf(w, "%s\t%s\t%s\n",
		keySprint("Key"),
		valueSprint("Value"),
		"Description") // plain text due to formatting/spacing issue
	fmt.Fprintf(w, "%s\t%s\t%s\n",
		keySprint(strings.Repeat("-", 20)),
		valueSprint(strings.Repeat("-", 15)),
		strings.Repeat("-", 40)) // plain text due to formatting/spacing issue
}

// printConfigRow prints a single configuration row with proper formatting
func printConfigRow(w io.Writer, style *OutputStyle, key, value, description string) {
	keySprint := style.KeyColor.SprintFunc()
	valueSprint := style.ValueColor.SprintFunc()

	if !style.EnableColors {
		keySprint = fmt.Sprint
		valueSprint = fmt.Sprint
	}

	// truncate long descriptions
	if len(description) > 50 {
		description = description[:47] + "..."
	}

	// truncate long values
	if len(value) > 25 {
		value = value[:22] + "..."
	}

	fmt.Fprintf(w, "%s\t%s\t%s\n",
		keySprint(key),
		valueSprint(value),
		description)
}

// printCanonicalSectionHeader prints a section header for canonical view (no description column)
func printCanonicalSectionHeader(w io.Writer, style *OutputStyle, groupName string) {
	groupSprint := style.GroupColor.SprintFunc()
	if !style.EnableColors {
		groupSprint = fmt.Sprint
	}

	fmt.Fprintf(w, "%s\n", groupSprint(fmt.Sprintf("▶ %s", groupName)))
	fmt.Fprintf(w, "%s\t%s\n",
		groupSprint("Key"),
		groupSprint("Value"))
	fmt.Fprintf(w, "%s\t%s\n",
		groupSprint(strings.Repeat("-", 30)),
		groupSprint(strings.Repeat("-", 45)))
}

// printCanonicalConfigRow prints a single configuration row for canonical view (no description)
func printCanonicalConfigRow(w io.Writer, style *OutputStyle, key, value string) {
	keySprint := style.KeyColor.SprintFunc()
	valueSprint := style.ValueColor.SprintFunc()

	if !style.EnableColors {
		keySprint = fmt.Sprint
		valueSprint = fmt.Sprint
	}

	fmt.Fprintf(w, "%s\t%s\n",
		keySprint(key),
		valueSprint(value))
}

// getConfigValue retrieves the current value for a configuration key using Viper
func getConfigValue(canonicalPath string) string {
	value := state.manager.Viper().Get(canonicalPath)

	// handle  nil values
	if value == nil {
		return "<not set>"
	}

	// handle empty strings
	if str, ok := value.(string); ok && str == "" {
		return "<not set>"
	}

	// basic formatting
	result := fmt.Sprintf("%v", value)

	// truncate if too long for table display
	if len(result) > 40 {
		return result[:37] + "..."
	}

	return result
}

// maskSensitiveValue masks sensitive values (API keys, etc.)
func maskSensitiveValue(key, value string) string {
	if value == "<not set>" || value == "" {
		return "<not set>"
	}

	// check if the key contains sensitive indicators
	keyLower := strings.ToLower(key)
	if strings.Contains(keyLower, "api") || strings.Contains(keyLower, "key") {
		if len(value) <= 5 {
			return strings.Repeat("*", len(value))
		}
		return value[:5] + "..."
	}

	return value
}

// maskSensitiveValueCanonical masks sensitive values for canonical view (longer values allowed)
func maskSensitiveValueCanonical(key, value string) string {
	if value == "<not set>" || value == "" {
		return "<not set>"
	}

	// check if the key contains sensitive indicators
	keyLower := strings.ToLower(key)
	if strings.Contains(keyLower, "api") || strings.Contains(keyLower, "key") {
		// for API keys, mask with first 5 chars
		if len(value) <= 5 {
			return strings.Repeat("*", len(value))
		}
		return value[:5] + "..."
	}

	// else allow up to 40 characters (for canonical view only)
	if len(value) > 40 {
		return value[:37] + "..."
	}

	return value
}

func init() {
	configCmd.AddCommand(listConfigCmd)
	listConfigCmd.Flags().Bool("aliases", false, "Show aliases view (default behavior)")
	listConfigCmd.Flags().Bool("canonical", false, "Show canonical configuration paths")
	listConfigCmd.MarkFlagsMutuallyExclusive("aliases", "canonical")
}
