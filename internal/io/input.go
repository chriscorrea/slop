package io

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadInput consolidates text from stdin, context files, and CLI arguments
// the order is: stdin, --context files, then CLI args
func ReadInput(stdin *os.File, cliArgs []string, contextFiles []string) (string, error) {
	return ReadInputWithCommandContext(stdin, cliArgs, contextFiles, "")
}

// ReadInputWithCommandContext consolidates text from command context
// the order is commnad, stdin, context files, then CLI args
func ReadInputWithCommandContext(stdin *os.File, cliArgs []string, contextFiles []string, commandContext string) (string, error) {
	var builder strings.Builder // https://pkg.go.dev/strings#Builder
	var hasContent bool

	// 1: command context comes first (if provided)
	if commandContext != "" {
		trimmed := strings.TrimSpace(commandContext)
		if trimmed != "" {
			builder.WriteString(trimmed)
			hasContent = true
		}
	}

	// 2: read from stdin if available
	if stdin != nil {
		// check if stdin has data available
		stat, err := stdin.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to stat stdin: %w", err)
		}

		// check if stdin is a pipe or has data
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			stdinContent, err := io.ReadAll(stdin)
			if err != nil {
				return "", fmt.Errorf("failed to read from stdin: %w", err)
			}

			if len(stdinContent) > 0 {
				// trim trailing whitespace
				content := strings.TrimRight(string(stdinContent), "\r\n\t ")
				if content != "" {
					if hasContent {
						builder.WriteString("\n\n")
					}
					builder.WriteString(content)
					hasContent = true
				}
			}
		}
	}

	// 3: read content from context files
	for _, filePath := range contextFiles {
		if filePath == "" {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read context file %q: %w", filePath, err)
		}

		// trim trailing whitespace
		fileContent := strings.TrimRight(string(content), "\r\n\t ")
		if fileContent != "" {
			if hasContent {
				builder.WriteString("\n\n")
			}
			builder.WriteString(fileContent)
			hasContent = true
		}
	}

	// join CLI arguments with spaces
	if len(cliArgs) > 0 {
		cliContent := strings.Join(cliArgs, " ")
		if cliContent != "" {
			if hasContent {
				builder.WriteString("\n\n")
			}
			builder.WriteString(cliContent)
			// hasContent = true
		}
	}

	return builder.String(), nil
}
