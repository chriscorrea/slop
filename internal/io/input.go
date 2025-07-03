package io

import (
	"fmt"
	"io"
	"os"
	"strings"

	slopContext "slop/internal/context"
)

// StructuredInput represents input components separated for synthetic message history
type StructuredInput struct {
	CommandContext string
	StdinContent   string
	ContextFiles   []slopContext.ContextFile
	CLIArgs        string
}

// ReadInput returns structured input components for synthetic message history
func ReadInput(stdin *os.File, cliArgs []string, contextFiles []slopContext.ContextFile, commandContext string) (*StructuredInput, error) {
	var stdinContent string
	var cliArgsString string

	// read from stdin if available
	if stdin != nil {
		// check if stdin has data available
		stat, err := stdin.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat stdin: %w", err)
		}

		// check if stdin is a pipe or has data
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			stdinData, err := io.ReadAll(stdin)
			if err != nil {
				return nil, fmt.Errorf("failed to read from stdin: %w", err)
			}

			if len(stdinData) > 0 {
				// trim trailing whitespace
				stdinContent = strings.TrimRight(string(stdinData), "\r\n\t ")
			}
		}
	}

	// process CLI arguments
	if len(cliArgs) > 0 {
		cliArgsString = strings.Join(cliArgs, " ")
	}

	return &StructuredInput{
		CommandContext: commandContext,
		StdinContent:   stdinContent,
		ContextFiles:   contextFiles,
		CLIArgs:        cliArgsString,
	}, nil
}
