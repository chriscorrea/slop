package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ContextResult contains the result of context processing
type ContextResult struct {
	AllContextFiles []string
	CLIContextFiles []string
	CmdContextFiles []string
}

// ContextManager handles merging of context files from CLI flags and commands
type ContextManager interface {
	ProcessContext(cmd *cobra.Command, additionalContextFiles []string) (*ContextResult, error)
}

// DefaultContextManager implements ContextManager
type DefaultContextManager struct{}

// NewContextManager creates a new DefaultContextManager
func NewContextManager() *DefaultContextManager {
	return &DefaultContextManager{}
}

// ProcessContext merges context files from CLI flags and command context files
func (c *DefaultContextManager) ProcessContext(cmd *cobra.Command, additionalContextFiles []string) (*ContextResult, error) {
	// get context files from CLI flag
	cliContextFiles, err := cmd.Flags().GetStringSlice("context")
	if err != nil {
		return nil, fmt.Errorf("failed to get context flag: %w", err)
	}

	// merge CLI context files with command context files
	allContextFiles := make([]string, 0, len(cliContextFiles)+len(additionalContextFiles))
	allContextFiles = append(allContextFiles, cliContextFiles...)
	allContextFiles = append(allContextFiles, additionalContextFiles...)

	return &ContextResult{
		AllContextFiles: allContextFiles,
		CLIContextFiles: cliContextFiles,
		CmdContextFiles: additionalContextFiles,
	}, nil
}

// HasContextFiles returns true if any context files are present
func (c *ContextResult) HasContextFiles() bool {
	return len(c.AllContextFiles) > 0
}

// HasCLIContextFiles returns true if context files were provided via CLI flag
func (c *ContextResult) HasCLIContextFiles() bool {
	return len(c.CLIContextFiles) > 0
}

// HasCommandContextFiles returns true if context files were provided by the command
func (c *ContextResult) HasCommandContextFiles() bool {
	return len(c.CmdContextFiles) > 0
}
