package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	slopContext "slop/internal/context"
)

// ManifestManager handles .slop/context manifest for persistent project context
type ManifestManager struct {
	workingDir string
}

// NewManifestManager creates new manifest manager for given working directory
func NewManifestManager(workingDir string) *ManifestManager {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}
	return &ManifestManager{
		workingDir: workingDir,
	}
}

// FindManifest searches for a .slop/context manifest file in the current directory
// returns the path to the manifest file and the dir, or empty strings if not found
func (m *ManifestManager) FindManifest() (string, string, error) {
	// check if .slop/context exists in current directoryy
	manifestPath := filepath.Join(m.workingDir, ".slop", "context")
	if _, err := os.Stat(manifestPath); err == nil {
		return manifestPath, m.workingDir, nil
	}

	// possible future enhancement: traverse the directory tree
	/*
		currentDir := m.workingDir

		for {
			// check if .slop/context exists in current directory
			manifestPath := filepath.Join(currentDir, ".slop", "context")
			if _, err := os.Stat(manifestPath); err == nil {
				return manifestPath, currentDir, nil
			}

			// move up one level
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				// reached root directory
				break
			}
			currentDir = parent
		}
	*/

	return "", "", nil // not found
}

// LoadManifest reads and parses a manifest file
func (m *ManifestManager) LoadManifest(manifestPath string) ([]string, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	var paths []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// skip empty lines, allow # comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		paths = append(paths, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	return paths, nil
}

// SaveManifest writes to manifest file
func (m *ManifestManager) SaveManifest(manifestPath string, paths []string) error {
	// ensure .slop directory exists
	dir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create .slop directory: %w", err)
	}

	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// write header comment
	if _, err := writer.WriteString("# slop context manifest\n"); err != nil {
		return fmt.Errorf("failed to write manifest header: %w", err)
	}
	if _, err := writer.WriteString("# This file contains paths to context files for this project\n\n"); err != nil {
		return fmt.Errorf("failed to write manifest header: %w", err)
	}

	// write paths
	for _, path := range paths {
		if _, err := writer.WriteString(path + "\n"); err != nil {
			return fmt.Errorf("failed to write path to manifest: %w", err)
		}
	}

	return nil
}

// AddPaths adds new paths to the manifest file
func (m *ManifestManager) AddPaths(manifestPath string, newPaths []string) error {
	// load existing paths
	var existingPaths []string
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// manifest doesn't exist yet, start with empty list
		existingPaths = []string{}
	} else {
		// manifest exists, load it
		var err error
		existingPaths, err = m.LoadManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to load existing manifest: %w", err)
		}
	}

	// merge paths, avoiding duplicates
	pathSet := make(map[string]bool)
	for _, path := range existingPaths {
		pathSet[path] = true
	}

	var allPaths []string
	allPaths = append(allPaths, existingPaths...)

	for _, newPath := range newPaths {
		if !pathSet[newPath] {
			allPaths = append(allPaths, newPath)
			pathSet[newPath] = true
		}
	}

	return m.SaveManifest(manifestPath, allPaths)
}

// ClearManifest removes all paths from the manifest file
func (m *ManifestManager) ClearManifest(manifestPath string) error {
	return m.SaveManifest(manifestPath, []string{})
}

// LoadProjectContext discovers and loads context files from the manifest
func (m *ManifestManager) LoadProjectContext() ([]slopContext.ContextFile, error) {
	manifestPath, projectRoot, err := m.FindManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to find manifest: %w", err)
	}

	if manifestPath == "" {
		// no manifest found
		return []slopContext.ContextFile{}, nil
	}

	paths, err := m.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	var contextFiles []slopContext.ContextFile

	for _, path := range paths {
		// resolve relative paths against project root
		var fullPath string
		if filepath.IsAbs(path) {
			fullPath = path
		} else {
			fullPath = filepath.Join(projectRoot, path)
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			// print warning to stderr, let user know
			fmt.Fprintf(os.Stderr, "Warning: could not read context file %s: %v\n", fullPath, err)
			continue
		}

		// trim trailing whitespace
		fileContent := strings.TrimRight(string(content), "\r\n\t ")
		if fileContent != "" {
			contextFiles = append(contextFiles, slopContext.ContextFile{
				Path:    fullPath,
				Content: fileContent,
			})
		}
	}

	return contextFiles, nil
}

// GetManifestPath returns the path where a manifest would be created
func (m *ManifestManager) GetManifestPath() string {
	return filepath.Join(m.workingDir, ".slop", "context")
}
