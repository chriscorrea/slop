package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestNewContextManager tests the factory function
func TestNewContextManager(t *testing.T) {
	manager := NewContextManager()
	if manager == nil {
		t.Fatal("Expected NewContextManager to return a non-nil manager")
	}

	// verify it implements the ContextManager interface
	var _ ContextManager = manager
}

// TestProcessContext tests nearly all context processing scenarios
func TestProcessContext(t *testing.T) {
	tests := []struct {
		name            string
		cliContextFiles []string
		additionalFiles []string
		expectedAll     []string
		expectedCLI     []string
		expectedCmd     []string
		shouldError     bool
	}{
		{
			name:            "empty everything",
			cliContextFiles: []string{},
			additionalFiles: []string{},
			expectedAll:     []string{},
			expectedCLI:     []string{},
			expectedCmd:     []string{},
		},
		{
			name:            "Files",
			cliContextFiles: []string{"windmill.txt", "snowball.txt"},
			additionalFiles: []string{},
			expectedAll:     []string{"windmill.txt", "snowball.txt"},
			expectedCLI:     []string{"windmill.txt", "snowball.txt"},
			expectedCmd:     []string{},
		},
		{
			name:            "duplicate file names allowed",
			cliContextFiles: []string{"file1.txt"},
			additionalFiles: []string{"file1.txt"},
			expectedAll:     []string{"file1.txt", "file1.txt"},
			expectedCLI:     []string{"file1.txt"},
			expectedCmd:     []string{"file1.txt"},
		},
		{
			name:            "files with special characters",
			cliContextFiles: []string{"file with spaces.txt", "file-with-dashes.txt"},
			additionalFiles: []string{"file_with_underscores.txt"},
			expectedAll:     []string{"file with spaces.txt", "file-with-dashes.txt", "file_with_underscores.txt"},
			expectedCLI:     []string{"file with spaces.txt", "file-with-dashes.txt"},
			expectedCmd:     []string{"file_with_underscores.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create command with context flag
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().StringSlice("context", []string{}, "context files")

			// set CLI context files if any
			if len(tt.cliContextFiles) > 0 {
				err := cmd.Flags().Set("context", strings.Join(tt.cliContextFiles, ","))
				if err != nil {
					t.Fatalf("Failed to set context flag: %v", err)
				}
			}

			manager := NewContextManager()
			result, err := manager.ProcessContext(cmd, tt.additionalFiles)

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify results
			if !reflect.DeepEqual(result.AllContextFiles, tt.expectedAll) {
				t.Errorf("AllContextFiles: expected %v, got %v", tt.expectedAll, result.AllContextFiles)
			}
			if !reflect.DeepEqual(result.CLIContextFiles, tt.expectedCLI) {
				t.Errorf("CLIContextFiles: expected %v, got %v", tt.expectedCLI, result.CLIContextFiles)
			}
			if !reflect.DeepEqual(result.CmdContextFiles, tt.expectedCmd) {
				t.Errorf("CmdContextFiles: expected %v, got %v", tt.expectedCmd, result.CmdContextFiles)
			}
		})
	}
}

// TestProcessContext_FlagError tests error handling when flag doesn't exist
func TestProcessContext_FlagError(t *testing.T) {
	// create command without context flag
	cmd := &cobra.Command{Use: "test"}

	manager := NewContextManager()
	result, err := manager.ProcessContext(cmd, []string{"file.txt"})

	if err == nil {
		t.Fatal("Expected error when context flag is not defined")
	}
	if result != nil {
		t.Errorf("Expected nil result when error occurs, got %v", result)
	}
	if !strings.Contains(err.Error(), "failed to get context flag") {
		t.Errorf("Expected error about flag, got: %v", err)
	}
}

// TestProcessContext_NilCommand tests panic handling with nil command
func TestProcessContext_NilCommand(t *testing.T) {
	manager := NewContextManager()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic when passing nil command")
		}
	}()

	_ = manager.ProcessContext(nil, []string{"file.txt"})
	t.Fatal("Expected panic but function returned normally")
}

// TestContextResult_HelperMethods tests all Has* methods
func TestContextResult_HelperMethods(t *testing.T) {
	tests := []struct {
		name               string
		allFiles           []string
		cliFiles           []string
		cmdFiles           []string
		expectedHasContext bool
		expectedHasCLI     bool
		expectedHasCommand bool
	}{
		{
			name:               "no files",
			allFiles:           []string{},
			cliFiles:           []string{},
			cmdFiles:           []string{},
			expectedHasContext: false,
			expectedHasCLI:     false,
			expectedHasCommand: false,
		},
		{
			name:               "only CLI files",
			allFiles:           []string{"windmill.txt"},
			cliFiles:           []string{"windmill.txt"},
			cmdFiles:           []string{},
			expectedHasContext: true,
			expectedHasCLI:     true,
			expectedHasCommand: false,
		},
		{
			name:               "only command files",
			allFiles:           []string{"command1.txt"},
			cliFiles:           []string{},
			cmdFiles:           []string{"command1.txt"},
			expectedHasContext: true,
			expectedHasCLI:     false,
			expectedHasCommand: true,
		},
		{
			name:               "both CLI and command files",
			allFiles:           []string{"windmill.txt", "command1.txt"},
			cliFiles:           []string{"windmill.txt"},
			cmdFiles:           []string{"command1.txt"},
			expectedHasContext: true,
			expectedHasCLI:     true,
			expectedHasCommand: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ContextResult{
				AllContextFiles: tt.allFiles,
				CLIContextFiles: tt.cliFiles,
				CmdContextFiles: tt.cmdFiles,
			}

			if result.HasContextFiles() != tt.expectedHasContext {
				t.Errorf("HasContextFiles(): expected %v, got %v",
					tt.expectedHasContext, result.HasContextFiles())
			}
			if result.HasCLIContextFiles() != tt.expectedHasCLI {
				t.Errorf("HasCLIContextFiles(): expected %v, got %v",
					tt.expectedHasCLI, result.HasCLIContextFiles())
			}
			if result.HasCommandContextFiles() != tt.expectedHasCommand {
				t.Errorf("HasCommandContextFiles(): expected %v, got %v",
					tt.expectedHasCommand, result.HasCommandContextFiles())
			}
		})
	}
}
