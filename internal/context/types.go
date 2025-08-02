package context

import "github.com/chriscorrea/slop/internal/llm/common"

// ContextFile represents a single context file with its path and content
type ContextFile struct {
	Path    string
	Content string
}

// ContextItem represents a processed context item with type information
type ContextItem struct {
	Path     string           // file path
	Type     string           // "conversation" or "file"
	Messages []common.Message // for conversations
	Content  string           // for raw files
}

// ContextResult contains the result of context processing
type ContextResult struct {
	AllContextFiles []string
	CLIContextFiles []string
	CmdContextFiles []string
	// Structured context data for synthetic message history
	ContextFileContents []ContextFile
	// Enhanced structured context data with conversation detection
	ProcessedItems []ContextItem
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

// HasStructuredContent returns true if structured context content is available
func (c *ContextResult) HasStructuredContent() bool {
	return len(c.ContextFileContents) > 0
}
