package cmd

import (
	"fmt"
	"os"
	"strings"

	slopContext "slop/internal/context"
	"slop/internal/manifest"

	"github.com/spf13/cobra"
)

// ContextManager handles merging of context files from CLI flags and commands
type ContextManager interface {
	ProcessContext(cmd *cobra.Command, additionalContextFiles []string) (*slopContext.ContextResult, error)
	ProcessContextWithFlags(cmd *cobra.Command, additionalContextFiles []string, skipProjectContext bool) (*slopContext.ContextResult, error)
}

// DefaultContextManager implements ContextManager
type DefaultContextManager struct{}

// NewContextManager creates a new DefaultContextManager
func NewContextManager() *DefaultContextManager {
	return &DefaultContextManager{}
}

// ProcessContext merges context files from CLI flags and command context files
func (c *DefaultContextManager) ProcessContext(cmd *cobra.Command, additionalContextFiles []string) (*slopContext.ContextResult, error) {
	return c.ProcessContextWithFlags(cmd, additionalContextFiles, false)
}

// ProcessContextWithFlags merges context files from CLI flags, command context files, and optionally project context
func (c *DefaultContextManager) ProcessContextWithFlags(cmd *cobra.Command, additionalContextFiles []string, skipProjectContext bool) (*slopContext.ContextResult, error) {
	// get context files from CLI flag
	cliContextFiles, err := cmd.Flags().GetStringSlice("context")
	if err != nil {
		return nil, fmt.Errorf("failed to get context flag: %w", err)
	}

	// load project context files if not skipped
	var projectContextFiles []slopContext.ContextFile
	if !skipProjectContext {
		manager := manifest.NewManifestManager("")
		projectContextFiles, err = manager.LoadProjectContext()
		if err != nil {
			return nil, fmt.Errorf("failed to load project context: %w", err)
		}
	}

	// merge CLI context files with command context files
	allContextFiles := make([]string, 0, len(cliContextFiles)+len(additionalContextFiles))
	allContextFiles = append(allContextFiles, cliContextFiles...)
	allContextFiles = append(allContextFiles, additionalContextFiles...)

	// read content from CLI and command context files for structured processing
	contextFileContents := make([]slopContext.ContextFile, 0, len(allContextFiles)+len(projectContextFiles))

	// add project context files first (they come before CLI context files)
	contextFileContents = append(contextFileContents, projectContextFiles...)

	// then add CLI and command context files
	for _, filePath := range allContextFiles {
		if filePath == "" {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read context file %q: %w", filePath, err)
		}

		// trim trailing whitespace (consistent with existing behavior)
		fileContent := strings.TrimRight(string(content), "\r\n\t ")
		if fileContent != "" {
			contextFileContents = append(contextFileContents, slopContext.ContextFile{
				Path:    filePath,
				Content: fileContent,
			})
		}
	}

	return &slopContext.ContextResult{
		AllContextFiles:     allContextFiles,
		CLIContextFiles:     cliContextFiles,
		CmdContextFiles:     additionalContextFiles,
		ContextFileContents: contextFileContents,
	}, nil
}
