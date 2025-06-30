package verbose

import (
	"bytes"
	"strings"
	"testing"

	"slop/internal/config"

	"github.com/fatih/color"
)

func TestPrintLLMParameters(t *testing.T) {
	// config with standard parameters for testing
	cfg := &config.Config{
		Parameters: config.Parameters{
			Temperature: 0.77,
			TopP:        0.99,
			MaxTokens:   2048,
		},
	}

	t.Run("DefaultOutput", func(t *testing.T) {
		var buf bytes.Buffer
		outputCfg := DefaultOutputConfig(&buf)

		PrintLLMParameters(cfg, "Hognitive Labs", "oink3-2025-07-05", outputCfg)

		output := buf.String()
		expectedStrings := []string{
			"Provider", "Hognitive Labs",
			"Model", "oink3-2025-07-05",
			"Temperature", "0.77",
			"Top P", "0.99",
			"Max Output Tokens", "2048",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(output, expected) {
				t.Errorf("Expected output to contain %q, got: %s", expected, output)
			}
		}
	})

	t.Run("WithoutColors", func(t *testing.T) {
		var buf bytes.Buffer
		outputCfg := DefaultOutputConfig(&buf)
		outputCfg.EnableColors = false

		PrintLLMParameters(cfg, "Entropic", "entropic-4-sestina-4-20250514", outputCfg)

		output := buf.String()
		// check for plain text without ANSI color codes
		if strings.Contains(output, "\x1b[") {
			t.Errorf("Expected output without color codes, got: %s", output)
		}

		expectedStrings := []string{
			"Provider:", "Entropic",
			"Model:", "entropic-4-sestina-4-20250514",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(output, expected) {
				t.Errorf("Expected output to contain %q, got: %s", expected, output)
			}
		}
	})

	t.Run("ShortSystemPrompt", func(t *testing.T) {
		var buf bytes.Buffer
		outputCfg := DefaultOutputConfig(&buf)

		cfgWithPrompt := *cfg
		cfgWithPrompt.Parameters.SystemPrompt = "You are a stochastic parrot."

		PrintLLMParameters(&cfgWithPrompt, "Hognitive Labs", "oink3-2025-07-05", outputCfg)

		output := buf.String()
		if !strings.Contains(output, "System Prompt") || !strings.Contains(output, "You are a stochastic parrot.") {
			t.Errorf("Expected output to contain system prompt, got: %s", output)
		}
	})

	t.Run("WithCustomColors", func(t *testing.T) {
		// force colors to be enabled for testing
		color.NoColor = false
		defer func() {
			// cleanup color state after test
			color.NoColor = false
		}()

		var buf bytes.Buffer
		customOutputCfg := &OutputConfig{
			Writer:       &buf,
			KeyColor:     color.New(color.FgRed),
			ValueColor:   color.New(color.FgBlue),
			HeaderColor:  color.New(color.FgGreen),
			EnableColors: true,
		}

		PrintLLMParameters(cfg, "Entropic", "entropic-4-sestina-4-20250514", customOutputCfg)

		output := buf.String()
		// verify output was generated
		if !strings.Contains(output, "Provider") || !strings.Contains(output, "Entropic") {
			t.Errorf("Expected output to contain provider information, got: %s", output)
		}

		// verify color codes are present when colors are enabled
		if !strings.Contains(output, "\x1b[") {
			t.Errorf("Expected output to contain color codes, got: %s", output)
		}
	})

	t.Run("WithNilOutputConfig", func(t *testing.T) {
		// this only tests that no panic occurs when outputCfg is nil
		PrintLLMParameters(cfg, "Ollama", "smollm2:latest", nil)
	})

	t.Run("OddNumberOfParameters", func(t *testing.T) {
		// create config with an odd number of parameters to test
		// that the last parameter displays correctly in its own row
		var buf bytes.Buffer
		outputCfg := DefaultOutputConfig(&buf)

		PrintLLMParameters(cfg, "Ollama", "smollm2:latest", outputCfg)

		output := buf.String()
		// check that the last parameter is on its own line
		lastParam := "Max Output Tokens"

		// split output by lines to check formatting
		lines := strings.Split(output, "\n")
		lastParamFound := false

		for _, line := range lines {
			if strings.Contains(line, lastParam) {
				// This line should not contain another parameter's key
				otherParams := []string{"Provider", "Model", "Temperature", "Top P"}
				for _, param := range otherParams {
					if strings.Contains(line, param) && param != lastParam {
						t.Errorf("Last parameter should be on its own line, found: %s", line)
					}
				}
				lastParamFound = true
				break
			}
		}

		if !lastParamFound {
			t.Errorf("Did not find last parameter %q in output: %s", lastParam, output)
		}
	})
}
