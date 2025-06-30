package io

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadInput(t *testing.T) {
	tests := []struct {
		name           string
		stdinContent   string
		cliArgs        []string
		contextFiles   []string
		fileContents   map[string]string // filename -> content
		expectedOutput string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "CLI args only",
			stdinContent:   "",
			cliArgs:        []string{"hello", "world"},
			contextFiles:   []string{},
			fileContents:   map[string]string{},
			expectedOutput: "hello world",
			expectError:    false,
		},
		{
			name:           "Empty CLI args",
			stdinContent:   "",
			cliArgs:        []string{},
			contextFiles:   []string{},
			fileContents:   map[string]string{},
			expectedOutput: "",
			expectError:    false,
		},
		{
			name:           "Stdin only",
			stdinContent:   "This is from stdin",
			cliArgs:        []string{},
			contextFiles:   []string{},
			fileContents:   map[string]string{},
			expectedOutput: "This is from stdin",
			expectError:    false,
		},
		{
			name:           "Context file only",
			stdinContent:   "",
			cliArgs:        []string{},
			contextFiles:   []string{"file1.txt"},
			fileContents:   map[string]string{"file1.txt": "Content from file 1"},
			expectedOutput: "Content from file 1",
			expectError:    false,
		},
		{
			name:           "Multiple context files",
			stdinContent:   "",
			cliArgs:        []string{},
			contextFiles:   []string{"file1.txt", "file2.txt"},
			fileContents:   map[string]string{"file1.txt": "Content from file 1", "file2.txt": "Content from file 2"},
			expectedOutput: "Content from file 1\n\nContent from file 2",
			expectError:    false,
		},
		{
			name:           "All sources combined",
			stdinContent:   "Stdin content",
			cliArgs:        []string{"cli", "args"},
			contextFiles:   []string{"file1.txt"},
			fileContents:   map[string]string{"file1.txt": "File content"},
			expectedOutput: "Stdin content\n\nFile content\n\ncli args",
			expectError:    false,
		},
		{
			name:           "Stdin with trailing whitespace",
			stdinContent:   "Stdin content\n\n\t  ",
			cliArgs:        []string{},
			contextFiles:   []string{},
			fileContents:   map[string]string{},
			expectedOutput: "Stdin content",
			expectError:    false,
		},
		{
			name:           "File with trailing whitespace",
			stdinContent:   "",
			cliArgs:        []string{},
			contextFiles:   []string{"file1.txt"},
			fileContents:   map[string]string{"file1.txt": "File content\n\r\n\t  "},
			expectedOutput: "File content",
			expectError:    false,
		},
		{
			name:           "Empty context files are ignored",
			stdinContent:   "",
			cliArgs:        []string{"args"},
			contextFiles:   []string{"", "file1.txt", ""},
			fileContents:   map[string]string{"file1.txt": "File content"},
			expectedOutput: "File content\n\nargs",
			expectError:    false,
		},
		{
			name:           "Empty file content",
			stdinContent:   "",
			cliArgs:        []string{"args"},
			contextFiles:   []string{"empty.txt"},
			fileContents:   map[string]string{"empty.txt": ""},
			expectedOutput: "args",
			expectError:    false,
		},
		{
			name:           "File content with only whitespace",
			stdinContent:   "",
			cliArgs:        []string{"args"},
			contextFiles:   []string{"whitespace.txt"},
			fileContents:   map[string]string{"whitespace.txt": "\n\t  \r\n"},
			expectedOutput: "args",
			expectError:    false,
		},
		{
			name:           "Nonexistent context file",
			stdinContent:   "",
			cliArgs:        []string{},
			contextFiles:   []string{"nonexistent.txt"},
			fileContents:   map[string]string{},
			expectedOutput: "",
			expectError:    true,
			errorContains:  "failed to read context file",
		},
		{
			name:           "Multiline content preservation",
			stdinContent:   "Line 1\nLine 2\nLine 3",
			cliArgs:        []string{"final", "args"},
			contextFiles:   []string{"multiline.txt"},
			fileContents:   map[string]string{"multiline.txt": "File line 1\nFile line 2"},
			expectedOutput: "Line 1\nLine 2\nLine 3\n\nFile line 1\nFile line 2\n\nfinal args",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create temp directory for test files
			tempDir := t.TempDir()

			// create context files in temp directory
			var contextFilePaths []string
			for _, fileName := range tt.contextFiles {
				if fileName == "" {
					contextFilePaths = append(contextFilePaths, "")
					continue
				}

				filePath := filepath.Join(tempDir, fileName)
				content, exists := tt.fileContents[fileName]
				if exists {
					err := os.WriteFile(filePath, []byte(content), 0644)
					if err != nil {
						t.Fatalf("Failed to create test file %s: %v", filePath, err)
					}
				}
				contextFilePaths = append(contextFilePaths, filePath)
			}

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
					w.WriteString(tt.stdinContent)
				}()

				stdin = r
			}

			// execute the function
			result, err := ReadInput(stdin, tt.cliArgs, contextFilePaths)

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

			// check output
			if result != tt.expectedOutput {
				t.Errorf("Expected output:\n%q\nGot:\n%q", tt.expectedOutput, result)
			}
		})
	}
}

func TestReadInput_NilStdin(t *testing.T) {
	// test with nil stdin (should not crash)
	result, err := ReadInput(nil, []string{"test", "args"}, []string{})
	if err != nil {
		t.Errorf("Expected no error with nil stdin, got: %v", err)
	}

	expected := "test args"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
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

	result, err := ReadInput(tempFile, []string{"test", "args"}, []string{})
	if err != nil {
		t.Errorf("Expected no error with empty stdin file, got: %v", err)
	}

	expected := "test args"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
