package verbose

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/fatih/color"
)

// OutputConfig contains parameters for verbose output formatting
type OutputConfig struct {
	Writer       io.Writer
	KeyColor     *color.Color
	ValueColor   *color.Color
	HeaderColor  *color.Color
	EnableColors bool
}

// DefaultOutputConfig returns a default configuration for verbose output
func DefaultOutputConfig(writer io.Writer) *OutputConfig {
	return &OutputConfig{
		Writer:       writer,
		KeyColor:     color.New(color.FgCyan, color.Bold),
		ValueColor:   color.New(color.FgMagenta),
		HeaderColor:  color.New(color.FgYellow, color.Bold),
		EnableColors: true,
	}
}

// PrintLLMParameters displays LLM parameters in a formatted, multi-column table.
func PrintLLMParameters(cfg *config.Config, providerName, modelName string, outputCfg *OutputConfig) {
	if outputCfg == nil {
		outputCfg = DefaultOutputConfig(os.Stderr) // Default to Stderr
	}

	w := tabwriter.NewWriter(outputCfg.Writer, 0, 0, 3, ' ', 0) // Increased padding

	// gather all parameters into a key value slice
	type param struct {
		Key   string
		Value string
	}

	params := []param{
		{Key: "Provider", Value: providerName},
		{Key: "Model", Value: modelName},
		{Key: "Temperature", Value: fmt.Sprintf("%.2f", cfg.Parameters.Temperature)},
		{Key: "Top P", Value: fmt.Sprintf("%.2f", cfg.Parameters.TopP)},
		{Key: "Max Output Tokens", Value: fmt.Sprintf("%d", cfg.Parameters.MaxTokens)},
	}

	// optinoally, add seed if it's configured
	if cfg.Parameters.Seed != nil {
		params = append(params, param{Key: "Seed", Value: fmt.Sprintf("%d", *cfg.Parameters.Seed)})
	}

	// iterate through the params slice and print rows in pairs
	for i := 0; i < len(params); i += 2 {
		p1 := params[i]

		// check if a second parameter exists for this row
		if (i + 1) < len(params) {
			p2 := params[i+1]
			printRow(w, outputCfg, p1.Key, p1.Value, p2.Key, p2.Value)
		} else {
			// If there's an odd number of params, print the last one on its own
			printRow(w, outputCfg, p1.Key, p1.Value, "", "")
		}
	}

	// add system prompt at end
	if cfg.Parameters.SystemPrompt != "" {

		sysPrompt := cfg.Parameters.SystemPrompt

		if len(sysPrompt) > 65 {
			sysPrompt = sysPrompt[:62] + "..."
		}

		printRow(w, outputCfg, "System Prompt", sysPrompt, "", "")

	}

	fmt.Fprintf(w, "\n") // add a final newline for spacing
	w.Flush()            // flush to write the aligned content
}

// printRow prints a multi-column row for one or two key-value pairs
// and handles color formatting and alignment via tabwriter
func printRow(w io.Writer, outputCfg *OutputConfig, key1, value1, key2, value2 string) {
	keySprint := outputCfg.KeyColor.SprintFunc()
	valueSprint := outputCfg.ValueColor.SprintFunc()

	// handle color enabling
	if !outputCfg.EnableColors {
		keySprint = fmt.Sprint
		valueSprint = fmt.Sprint
	}

	if key2 != "" {
		// full, four-column row for two values
		fmt.Fprintf(w, "%s:\t%s\t%s:\t%s\n",
			keySprint(key1),
			valueSprint(value1),
			keySprint(key2),
			valueSprint(value2),
		)
	} else {
		// this is a row with only two columns (for one last item in odd-numbered list)
		fmt.Fprintf(w, "%s:\t%s\n",
			keySprint(key1),
			valueSprint(value1),
		)
	}
}
