package io

import (
	"os"
	"strings"
	"testing"

	slopContext "slop/internal/context"
)

func TestReadInput_StructuredProcessing(t *testing.T) {
	tests := []struct {
		name              string
		stdinContent      string
		cliArgs           []string
		contextFiles      []slopContext.ContextFile
		commandContext    string
		expectedStdin     string
		expectedCLI       string
		expectedCommand   string
		expectedFileCount int
		expectError       bool
		errorContains     string
	}{
		{
			name:              "CLI args only",
			stdinContent:      "",
			cliArgs:           []string{"hello", "world"},
			contextFiles:      []slopContext.ContextFile{},
			commandContext:    "",
			expectedStdin:     "",
			expectedCLI:       "hello world",
			expectedCommand:   "",
			expectedFileCount: 0,
			expectError:       false,
		},
		{
			name:              "Empty CLI args",
			stdinContent:      "",
			cliArgs:           []string{},
			contextFiles:      []slopContext.ContextFile{},
			commandContext:    "",
			expectedStdin:     "",
			expectedCLI:       "",
			expectedCommand:   "",
			expectedFileCount: 0,
			expectError:       false,
		},
		{
			name:              "Stdin only",
			stdinContent:      "This is from stdin",
			cliArgs:           []string{},
			contextFiles:      []slopContext.ContextFile{},
			commandContext:    "",
			expectedStdin:     "This is from stdin",
			expectedCLI:       "",
			expectedCommand:   "",
			expectedFileCount: 0,
			expectError:       false,
		},
		{
			name:         "Context files",
			stdinContent: "",
			cliArgs:      []string{},
			contextFiles: []slopContext.ContextFile{
				{Path: "file1.txt", Content: "Content from file 1"},
				{Path: "file2.txt", Content: "Content from file 2"},
			},
			commandContext:    "",
			expectedStdin:     "",
			expectedCLI:       "",
			expectedCommand:   "",
			expectedFileCount: 2,
			expectError:       false,
		},
		{
			name:              "Command context only",
			stdinContent:      "",
			cliArgs:           []string{},
			contextFiles:      []slopContext.ContextFile{},
			commandContext:    "This is command context",
			expectedStdin:     "",
			expectedCLI:       "",
			expectedCommand:   "This is command context",
			expectedFileCount: 0,
			expectError:       false,
		},
		{
			name:         "All sources combined",
			stdinContent: "Stdin content",
			cliArgs:      []string{"cli", "args"},
			contextFiles: []slopContext.ContextFile{
				{Path: "file1.txt", Content: "File content"},
			},
			commandContext:    "Command context",
			expectedStdin:     "Stdin content",
			expectedCLI:       "cli args",
			expectedCommand:   "Command context",
			expectedFileCount: 1,
			expectError:       false,
		},
		{
			name:              "Stdin with trailing whitespace",
			stdinContent:      "Stdin content\n\n\t  ",
			cliArgs:           []string{},
			contextFiles:      []slopContext.ContextFile{},
			commandContext:    "",
			expectedStdin:     "Stdin content",
			expectedCLI:       "",
			expectedCommand:   "",
			expectedFileCount: 0,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create stdin pipe if needed
			var stdin *os.File
			if tt.stdinContent != "" {
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatalf("Failed to create pipe: %v", err)
				}
				defer r.Close()
				defer w.Close()

				// Write content to pipe
				go func() {
					defer w.Close()
					_, _ = w.WriteString(tt.stdinContent)
				}()

				stdin = r
			}

			// execute the function
			result, err := ReadInput(stdin, tt.cliArgs, tt.contextFiles, tt.commandContext)

			// check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
				return
			}

			// check structured output
			if result.CommandContext != tt.expectedCommand {
				t.Errorf("Expected command context:\n%q\nGot:\n%q", tt.expectedCommand, result.CommandContext)
			}
			if result.StdinContent != tt.expectedStdin {
				t.Errorf("Expected stdin content:\n%q\nGot:\n%q", tt.expectedStdin, result.StdinContent)
			}
			if result.CLIArgs != tt.expectedCLI {
				t.Errorf("Expected CLI args:\n%q\nGot:\n%q", tt.expectedCLI, result.CLIArgs)
			}
			if len(result.ContextFiles) != tt.expectedFileCount {
				t.Errorf("Expected %d context files, got %d", tt.expectedFileCount, len(result.ContextFiles))
			}

			// verify context file content matches
			for i, expectedFile := range tt.contextFiles {
				if i >= len(result.ContextFiles) {
					t.Errorf("Missing expected context file %d", i)
					continue
				}
				if result.ContextFiles[i].Path != expectedFile.Path {
					t.Errorf("Expected context file path %q, got %q", expectedFile.Path, result.ContextFiles[i].Path)
				}
				if result.ContextFiles[i].Content != expectedFile.Content {
					t.Errorf("Expected context file content %q, got %q", expectedFile.Content, result.ContextFiles[i].Content)
				}
			}
		})
	}
}

func TestReadInput_NilStdin(t *testing.T) {
	// test with nil stdin (should not crash)
	result, err := ReadInput(nil, []string{"test", "args"}, []slopContext.ContextFile{}, "")
	if err != nil {
		t.Errorf("Expected no error with nil stdin, got: %v", err)
	}

	expected := "test args"
	if result.CLIArgs != expected {
		t.Errorf("Expected CLI args %q, got %q", expected, result.CLIArgs)
	}
	if result.StdinContent != "" {
		t.Errorf("Expected empty stdin content, got %q", result.StdinContent)
	}
}

func TestReadInput_StdinWithoutData(t *testing.T) {
	// test with stdin that has no data (regular file/terminal)
	// simulates running the command interactively
	tempFile, err := os.CreateTemp("", "test_stdin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	result, err := ReadInput(tempFile, []string{"test", "args"}, []slopContext.ContextFile{}, "")
	if err != nil {
		t.Errorf("Expected no error with empty stdin file, got: %v", err)
	}

	expected := "test args"
	if result.CLIArgs != expected {
		t.Errorf("Expected CLI args %q, got %q", expected, result.CLIArgs)
	}
	if result.StdinContent != "" {
		t.Errorf("Expected empty stdin content, got %q", result.StdinContent)
	}
}
