package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	slopContext "github.com/chriscorrea/slop/internal/context"

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

// TestProcessContext tests context processing scenarios
func TestProcessContext(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "slop-context-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create minimal test files
	testFiles := []string{"windmill.txt", "snowball.txt", "file1.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	tests := []struct {
		name            string
		cliContextFiles []string
		additionalFiles []string
		expectedAll     []string
		expectedCLI     []string
		expectedCmd     []string
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
			name:            "cli files only",
			cliContextFiles: []string{filepath.Join(tempDir, "windmill.txt"), filepath.Join(tempDir, "snowball.txt")},
			additionalFiles: []string{},
			expectedAll:     []string{filepath.Join(tempDir, "windmill.txt"), filepath.Join(tempDir, "snowball.txt")},
			expectedCLI:     []string{filepath.Join(tempDir, "windmill.txt"), filepath.Join(tempDir, "snowball.txt")},
			expectedCmd:     []string{},
		},
		{
			name:            "duplicate files allowed",
			cliContextFiles: []string{filepath.Join(tempDir, "file1.txt")},
			additionalFiles: []string{filepath.Join(tempDir, "file1.txt")},
			expectedAll:     []string{filepath.Join(tempDir, "file1.txt"), filepath.Join(tempDir, "file1.txt")},
			expectedCLI:     []string{filepath.Join(tempDir, "file1.txt")},
			expectedCmd:     []string{filepath.Join(tempDir, "file1.txt")},
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
			// Use ProcessContextWithFlags with skipProjectContext=true to avoid file reading in tests
			result, err := manager.ProcessContextWithFlags(cmd, tt.additionalFiles, true)

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

// TestProcessContext_ErrorCases tests error handling scenarios
func TestProcessContext_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		setupCmd    func() *cobra.Command
		expectPanic bool
		expectError bool
		errorText   string
	}{
		{
			name: "missing context flag",
			setupCmd: func() *cobra.Command {
				return &cobra.Command{Use: "test"} // no context flag defined
			},
			expectError: true,
			errorText:   "failed to get context flag",
		},
		{
			name: "nil command",
			setupCmd: func() *cobra.Command {
				return nil
			},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewContextManager()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("Expected panic but function returned normally")
					}
				}()
				_, _ = manager.ProcessContext(tt.setupCmd(), []string{"file.txt"})
				t.Fatal("Expected panic but function returned normally")
				return
			}

			result, err := manager.ProcessContext(tt.setupCmd(), []string{"file.txt"})

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if result != nil {
					t.Errorf("Expected nil result when error occurs, got %v", result)
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorText, err)
				}
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
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
			result := &slopContext.ContextResult{
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
