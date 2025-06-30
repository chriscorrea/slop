package data

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// see https://pkg.go.dev/embed for more on embedding files

//go:embed configs/*.json
var configFS embed.FS

// ModelInfo represents model configuration for a provider
type ModelInfo struct {
	Fast string `json:"fast"`
	Deep string `json:"deep"`
}

// ProviderInfo represents a provider's configuration information
type ProviderInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Reference   string    `json:"reference,omitempty"`
	Models      ModelInfo `json:"models"`
}

// ProvidersData represents the structure of models.json
type ProvidersData struct {
	Providers map[string]ProviderInfo `json:"providers"`
}

// ProviderRegistry handles loading and accessing provider data
type ProviderRegistry struct {
	data *ProvidersData
}

// NewProviderRegistry creates a new provider registry instance
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{}
}

// Load loads provider data from models.json with intelligent path discovery
func (p *ProviderRegistry) Load() error {
	data, err := loadProvidersData()
	if err != nil {
		return err
	}
	p.data = data
	return nil
}

// GetProviders returns all available providers
func (p *ProviderRegistry) GetProviders() map[string]ProviderInfo {
	if p.data == nil {
		return nil
	}
	return p.data.Providers
}

// GetProvider returns a specific provider by key
func (p *ProviderRegistry) GetProvider(key string) (ProviderInfo, bool) {
	if p.data == nil {
		return ProviderInfo{}, false
	}
	provider, exists := p.data.Providers[key]
	return provider, exists
}

// GetRemoteProviders returns providers suitable for remote use (excludes local-only providers)
func (p *ProviderRegistry) GetRemoteProviders() map[string]ProviderInfo {
	if p.data == nil {
		return nil
	}

	remote := make(map[string]ProviderInfo)
	for key, info := range p.data.Providers {
		// skip ollama as it's local-only
		if key != "ollama" {
			remote[key] = info
		}
	}
	return remote
}

// GetProviderOptions returns formatted options for survey selection
func (p *ProviderRegistry) GetProviderOptions() []string {
	remote := p.GetRemoteProviders()
	var options []string

	for _, info := range remote {
		option := fmt.Sprintf("%s - %s", info.Name, info.Description)
		options = append(options, option)
	}
	return options
}

// GetProviderKeyFromOption extracts the provider key from a formatted option string
func (p *ProviderRegistry) GetProviderKeyFromOption(selectedOption string) string {
	if p.data == nil {
		return ""
	}

	for key, info := range p.data.Providers {
		if selectedOption == fmt.Sprintf("%s - %s", info.Name, info.Description) {
			return key
		}
	}
	return ""
}

// loadProvidersData loads provider model data from models.json with embedded fallback
func loadProvidersData() (*ProvidersData, error) {
	// first try to load from embedded assets
	if embeddedData, err := configFS.ReadFile("configs/models.json"); err == nil {
		var providersData ProvidersData
		if err := json.Unmarshal(embeddedData, &providersData); err != nil {
			return nil, fmt.Errorf("failed to parse embedded models.json: %w", err)
		}
		return &providersData, nil
	}

	// fall back to external file locations
	var possiblePaths []string

	// get dir where the executable is located
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	possiblePaths = []string{
		filepath.Join(execDir, "configs", "models.json"), // Preferred: same dir as executable/configs/
		"configs/models.json",                            // Preferred: current working dir/configs/
		filepath.Join(execDir, "models.json"),            // Fallback: same dir as executable
		"models.json",                                    // Fallback: current working directory
	}

	var modelsPath string
	var data []byte

	for _, path := range possiblePaths {
		if fileData, err := os.ReadFile(path); err == nil {
			data = fileData
			modelsPath = path
			break
		}
	}

	if data == nil {
		return nil, fmt.Errorf("models.json not found in embedded assets or any of these locations: %v", possiblePaths)
	}

	var providersData ProvidersData
	if err := json.Unmarshal(data, &providersData); err != nil {
		return nil, fmt.Errorf("failed to parse models.json at %s: %w", modelsPath, err)
	}

	return &providersData, nil
}
