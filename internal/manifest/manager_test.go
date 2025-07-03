package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	slopContext "github.com/chriscorrea/slop/internal/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createManifestFile is a helper function to create a manifest file w/ content
func createManifestFile(t *testing.T, dir, content string) string {
	t.Helper()

	slopDir := filepath.Join(dir, ".slop")
	err := os.MkdirAll(slopDir, 0700)
	require.NoError(t, err)

	manifestPath := filepath.Join(slopDir, "context")
	err = os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	return manifestPath
}

// createTestFile is a helper function to create a test file w/ content
func createTestFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

func TestNewManifestManager(t *testing.T) {
	tests := []struct {
		name           string
		workingDir     string
		expectedResult func(string) string
	}{
		{
			name:       "with non-empty working directory",
			workingDir: "/test/path",
			expectedResult: func(string) string {
				return "/test/path"
			},
		},
		{
			name:       "with empty working directory",
			workingDir: "",
			expectedResult: func(string) string {
				wd, _ := os.Getwd()
				return wd
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManifestManager(tt.workingDir)

			assert.NotNil(t, manager)
			assert.Equal(t, tt.expectedResult(tt.workingDir), manager.workingDir)
		})
	}
}

func TestManifestManager_FindManifest(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) string
		expectedFound bool
		expectError   bool
	}{
		{
			name: "manifest exists in current directory",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				createManifestFile(t, tempDir, "file1.txt\nfile2.txt")
				return tempDir
			},
			expectedFound: true,
			expectError:   false,
		},
		{
			name: "manifest does not exist",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedFound: false,
			expectError:   false,
		},
		{
			name: "working directory does not exist",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			expectedFound: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir := tt.setupFunc(t)
			manager := NewManifestManager(workingDir)

			manifestPath, foundDir, err := manager.FindManifest()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedFound {
				assert.NotEmpty(t, manifestPath)
				assert.Equal(t, workingDir, foundDir)
			} else {
				assert.Empty(t, manifestPath)
				assert.Empty(t, foundDir)
			}
		})
	}
}

func TestManifestManager_LoadManifest(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedPaths []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "simple paths",
			content:       "file1.txt\nfile2.txt\npath/to/file3.txt",
			expectedPaths: []string{"file1.txt", "file2.txt", "path/to/file3.txt"},
		},
		{
			name:          "paths with comments and empty lines",
			content:       "# This is a comment\nfile1.txt\n\n# Another comment\nfile2.txt\n\n# Empty line above and below\n\npath/to/file3.txt\n# Final comment",
			expectedPaths: []string{"file1.txt", "file2.txt", "path/to/file3.txt"},
		},
		{
			name:          "paths with whitespace",
			content:       "  file1.txt  \n\t file2.txt \t\n   path/to/file3.txt   ",
			expectedPaths: []string{"file1.txt", "file2.txt", "path/to/file3.txt"},
		},
		{
			name:          "empty file",
			content:       "",
			expectedPaths: nil,
		},
		{
			name:          "only comments and whitespace",
			content:       "# comment1\n\n# comment2\n   \n\t\n   ",
			expectedPaths: nil,
		},
		{
			name:          "file not found",
			content:       "nonexistent",
			expectError:   true,
			errorContains: "failed to open manifest file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var manifestPath string

			if tt.name == "file not found" {
				manifestPath = filepath.Join(tempDir, "nonexistent.txt")
			} else {
				manifestPath = createManifestFile(t, tempDir, tt.content)
			}

			manager := NewManifestManager(tempDir)
			paths, err := manager.LoadManifest(manifestPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, paths)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPaths, paths)
			}
		})
	}
}

func TestManifestManager_SaveManifest(t *testing.T) {
	tests := []struct {
		name             string
		paths            []string
		createDir        bool
		verifyDirCreated bool
	}{
		{
			name:             "save with basic paths",
			paths:            []string{"muriel.txt", "boxer.txt", "path/to/raven.txt"},
			createDir:        false,
			verifyDirCreated: true,
		},
		{
			name:             "save empty paths list",
			paths:            []string{},
			createDir:        false,
			verifyDirCreated: true,
		},
		{
			name:      "save to existing directory",
			paths:     []string{"existing.txt"},
			createDir: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			manifestPath := filepath.Join(tempDir, ".slop", "context")

			if tt.createDir {
				err := os.MkdirAll(filepath.Dir(manifestPath), 0700)
				require.NoError(t, err)
			}

			manager := NewManifestManager("")
			err := manager.SaveManifest(manifestPath, tt.paths)

			assert.NoError(t, err)
			assert.FileExists(t, manifestPath)

			if tt.verifyDirCreated {
				assert.DirExists(t, filepath.Dir(manifestPath))
			}

			// verify content
			content, err := os.ReadFile(manifestPath)
			assert.NoError(t, err)

			lines := strings.Split(string(content), "\n")
			assert.Contains(t, lines[0], "# slop context manifest")
			assert.Contains(t, lines[1], "# This file contains paths")

			contentStr := string(content)
			for _, path := range tt.paths {
				assert.Contains(t, contentStr, path)
			}
		})
	}
}

func TestManifestManager_AddPaths(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		newPaths        []string
		expectedPaths   []string
	}{
		{
			name:            "add to new manifest",
			existingContent: "",
			newPaths:        []string{"file1.txt", "file2.txt"},
			expectedPaths:   []string{"file1.txt", "file2.txt"},
		},
		{
			name:            "add to existing manifest",
			existingContent: "existing1.txt\nexisting2.txt",
			newPaths:        []string{"new1.txt", "new2.txt"},
			expectedPaths:   []string{"existing1.txt", "existing2.txt", "new1.txt", "new2.txt"},
		},
		{
			name:            "add with duplicates (deduplication)",
			existingContent: "file1.txt\nfile2.txt",
			newPaths:        []string{"file1.txt", "file3.txt", "file2.txt", "file4.txt"},
			expectedPaths:   []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt"},
		},
		{
			name:            "add all duplicates",
			existingContent: "file1.txt\nfile2.txt",
			newPaths:        []string{"file1.txt", "file2.txt"},
			expectedPaths:   []string{"file1.txt", "file2.txt"},
		},
		{
			name:            "add empty paths list",
			existingContent: "existing.txt",
			newPaths:        []string{},
			expectedPaths:   []string{"existing.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var manifestPath string

			if tt.existingContent == "" {
				manifestPath = filepath.Join(tempDir, ".slop", "context")
			} else {
				manifestPath = createManifestFile(t, tempDir, tt.existingContent)
			}

			manager := NewManifestManager(tempDir)
			err := manager.AddPaths(manifestPath, tt.newPaths)

			assert.NoError(t, err)

			paths, err := manager.LoadManifest(manifestPath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestManifestManager_ClearManifest(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
	}{
		{
			name:            "clear existing manifest",
			existingContent: "file1.txt\nfile2.txt\nfile3.txt",
		},
		{
			name:            "clear non-existent manifest",
			existingContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var manifestPath string

			if tt.existingContent == "" {
				manifestPath = filepath.Join(tempDir, ".slop", "context")
			} else {
				manifestPath = createManifestFile(t, tempDir, tt.existingContent)
			}

			manager := NewManifestManager(tempDir)
			err := manager.ClearManifest(manifestPath)

			assert.NoError(t, err)
			assert.FileExists(t, manifestPath)

			paths, err := manager.LoadManifest(manifestPath)
			assert.NoError(t, err)
			assert.Empty(t, paths)
		})
	}
}

func TestManifestManager_LoadProjectContext(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) string
		expectedFiles int
		verifyContent func(t *testing.T, contextFiles []slopContext.ContextFile)
	}{
		{
			name: "success with multiple files",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				file1Path := filepath.Join(tempDir, "file1.txt")
				file2Path := filepath.Join(tempDir, "subdir", "file2.txt")
				file3Path := filepath.Join(tempDir, "file3.txt")

				createTestFile(t, file1Path, "Content of file 1")
				createTestFile(t, file2Path, "Content of file 2")
				createTestFile(t, file3Path, "Content of file 3\n\n\t  ")

				manifestContent := "file1.txt\nsubdir/file2.txt\nfile3.txt"
				createManifestFile(t, tempDir, manifestContent)
				return tempDir
			},
			expectedFiles: 3,
			verifyContent: func(t *testing.T, contextFiles []slopContext.ContextFile) {
				assert.Equal(t, "Content of file 1", contextFiles[0].Content)
				assert.Equal(t, "Content of file 2", contextFiles[1].Content)
				assert.Equal(t, "Content of file 3", contextFiles[2].Content)
			},
		},
		{
			name: "absolute and relative paths",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				absoluteFile := filepath.Join(os.TempDir(), "absolute_test.txt")
				createTestFile(t, absoluteFile, "Absolute file content")
				t.Cleanup(func() { os.Remove(absoluteFile) })

				relativeFile := filepath.Join(tempDir, "relative.txt")
				createTestFile(t, relativeFile, "Relative file content")

				manifestContent := absoluteFile + "\nrelative.txt"
				createManifestFile(t, tempDir, manifestContent)
				return tempDir
			},
			expectedFiles: 2,
			verifyContent: func(t *testing.T, contextFiles []slopContext.ContextFile) {
				assert.Equal(t, "Absolute file content", contextFiles[0].Content)
				assert.Equal(t, "Relative file content", contextFiles[1].Content)
			},
		},
		{
			name: "missing files are skipped",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				validFile := filepath.Join(tempDir, "valid.txt")
				createTestFile(t, validFile, "Valid content")

				manifestContent := "valid.txt\nmissing.txt\nvalid.txt"
				createManifestFile(t, tempDir, manifestContent)
				return tempDir
			},
			expectedFiles: 2,
			verifyContent: func(t *testing.T, contextFiles []slopContext.ContextFile) {
				assert.Equal(t, "Valid content", contextFiles[0].Content)
				assert.Equal(t, "Valid content", contextFiles[1].Content)
			},
		},
		{
			name: "empty files are skipped",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				emptyFile := filepath.Join(tempDir, "empty.txt")
				whitespaceFile := filepath.Join(tempDir, "whitespace.txt")
				validFile := filepath.Join(tempDir, "valid.txt")

				createTestFile(t, emptyFile, "")
				createTestFile(t, whitespaceFile, "   \n\t\r  ")
				createTestFile(t, validFile, "Valid content")

				manifestContent := "empty.txt\nwhitespace.txt\nvalid.txt"
				createManifestFile(t, tempDir, manifestContent)
				return tempDir
			},
			expectedFiles: 1,
			verifyContent: func(t *testing.T, contextFiles []slopContext.ContextFile) {
				assert.Equal(t, "Valid content", contextFiles[0].Content)
			},
		},
		{
			name: "no manifest file",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedFiles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir := tt.setupFunc(t)
			manager := NewManifestManager(workingDir)

			contextFiles, err := manager.LoadProjectContext()

			assert.NoError(t, err)
			assert.Len(t, contextFiles, tt.expectedFiles)

			if tt.verifyContent != nil && len(contextFiles) > 0 {
				tt.verifyContent(t, contextFiles)
			}
		})
	}
}

func TestManifestManager_GetManifestPath(t *testing.T) {
	tests := []struct {
		name       string
		workingDir string
		expected   func(string) string
	}{
		{
			name:       "with working directory",
			workingDir: "/test/path",
			expected: func(wd string) string {
				return filepath.Join(wd, ".slop", "context")
			},
		},
		{
			name:       "with empty working directory",
			workingDir: "",
			expected: func(wd string) string {
				cwd, _ := os.Getwd()
				return filepath.Join(cwd, ".slop", "context")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManifestManager(tt.workingDir)
			manifestPath := manager.GetManifestPath()

			expected := tt.expected(tt.workingDir)
			assert.Equal(t, expected, manifestPath)
		})
	}
}
