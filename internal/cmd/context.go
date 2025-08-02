package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	slopContext "github.com/chriscorrea/slop/internal/context"
	"github.com/chriscorrea/slop/internal/manifest"
	"github.com/chriscorrea/slop/internal/parser"

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
	processedItems := make([]slopContext.ContextItem, 0, len(allContextFiles)+len(projectContextFiles))

	// add project context files first (they come before CLI context files)
	contextFileContents = append(contextFileContents, projectContextFiles...)
	for _, contextFile := range projectContextFiles {
		processedItem := c.processContextFile(contextFile.Path, contextFile.Content, state.logger)
		processedItems = append(processedItems, processedItem)
	}

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

			// process with smart detection (for conversations vs other files)
			processedItem := c.processContextFile(filePath, fileContent, state.logger)
			processedItems = append(processedItems, processedItem)
		}
	}

	return &slopContext.ContextResult{
		AllContextFiles:     allContextFiles,
		CLIContextFiles:     cliContextFiles,
		CmdContextFiles:     additionalContextFiles,
		ContextFileContents: contextFileContents,
		ProcessedItems:      processedItems,
	}, nil
}

// processContextFile intelligently processes a context file, detecting conversations vs regular files
func (c *DefaultContextManager) processContextFile(path string, content string, logger *slog.Logger) slopContext.ContextItem {
	// try JSON parsing first
	if messages, err := parser.ParseJSONHistory([]byte(content)); err == nil {
		if logger != nil {
			logger.Debug("Context file detected as JSON conversation", "file", path, "messages", len(messages))
		}
		return slopContext.ContextItem{
			Path:     path,
			Type:     "conversation",
			Messages: messages,
		}
	}

	// try extension-based detection
	if parser.IsConversationFile(path) {
		if messages, err := parser.ParseTextHistory(content); err == nil {
			if logger != nil {
				logger.Debug("Context file detected as text conversation", "file", path, "messages", len(messages))
			}
			return slopContext.ContextItem{
				Path:     path,
				Type:     "conversation",
				Messages: messages,
			}
		} else if logger != nil {
			logger.Debug("Context file has conversation extension but failed parsing, treating as text", "file", path, "error", err)
		}
	}

	// fallback to regular file
	if logger != nil {
		logger.Debug("Context file treated as regular text", "file", path, "content_length", len(content))
	}
	return slopContext.ContextItem{
		Path:    path,
		Type:    "file",
		Content: content,
	}
}
