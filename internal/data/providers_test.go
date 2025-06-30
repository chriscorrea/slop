package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderRegistry(t *testing.T) {
	// create a models.json for testing
	tempDir := t.TempDir()
	configsDir := filepath.Join(tempDir, "configs")
	err := os.MkdirAll(configsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp configs dir: %v", err)
	}

	// mock models.json content (simplified for testing)
	modelsJSON := `{
		"providers": {
			"anthropic": {
				"name": "Anthropic",
				"description": "Claude models from Anthropic",
				"reference": "https://docs.anthropic.com/en/docs/about-claude/models/overview",
				"models": {
					"fast": "claude-3-5-haiku-latest",
					"deep": "claude-sonnet-4-20250514"
				}
			},
			"ollama": {
				"name": "Ollama",
				"description": "Local models via Ollama",
				"reference": "https://ollama.com/library",
				"models": {
					"fast": "gemma3n:latest",
					"deep": "deepseek-r1:14b"
				}
			}
		}
	}`

	modelsPath := filepath.Join(configsDir, "models.json")
	err = os.WriteFile(modelsPath, []byte(modelsJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test models.json: %v", err)
	}

	// change to temp directory so loader finds the test file
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// test the provider registry
	registry := NewProviderRegistry()

	t.Run("Load", func(t *testing.T) {
		err := registry.Load()
		if err != nil {
			t.Fatalf("Failed to load providers: %v", err)
		}
	})

	t.Run("GetProviders", func(t *testing.T) {
		providers := registry.GetProviders()
		if len(providers) < 2 {
			t.Errorf("Expected at least 2 providers, got %d", len(providers))
		}

		if _, exists := providers["anthropic"]; !exists {
			t.Error("Expected anthropic provider to exist")
		}

		if _, exists := providers["ollama"]; !exists {
			t.Error("Expected ollama provider to exist")
		}
	})

	t.Run("GetProvider", func(t *testing.T) {
		provider, exists := registry.GetProvider("anthropic")
		if !exists {
			t.Error("Expected anthropic provider to exist")
		}

		if provider.Name != "Anthropic" {
			t.Errorf("Expected provider name 'Anthropic', got '%s'", provider.Name)
		}

		if provider.Models.Fast == "" {
			t.Errorf("Expected fast model to be set, got empty string")
		}

		// error case - non-existent provider
		t.Run("NonExistentProvider", func(t *testing.T) {
			provider, exists := registry.GetProvider("non-existent-provider")
			if exists {
				t.Error("Expected non-existent provider to return false")
			}

			if provider.Name != "" || provider.Description != "" || provider.Models.Fast != "" {
				t.Errorf("Expected empty ProviderInfo for non-existent provider, got %+v", provider)
			}
		})
	})

	t.Run("GetRemoteProviders", func(t *testing.T) {
		remote := registry.GetRemoteProviders()
		if len(remote) < 1 {
			t.Errorf("Expected at least 1 remote provider, got %d", len(remote))
		}

		if _, exists := remote["anthropic"]; !exists {
			t.Error("Expected anthropic in remote providers")
		}

		if _, exists := remote["ollama"]; exists {
			t.Error("Expected ollama to be excluded from remote providers")
		}
	})
	t.Run("GetProviderKeyFromOption", func(t *testing.T) {
		option := "Anthropic - Claude models from Anthropic"
		key := registry.GetProviderKeyFromOption(option)
		if key != "anthropic" {
			t.Errorf("Expected key 'anthropic', got '%s'", key)
		}

		// invalid option string
		t.Run("InvalidOptionString", func(t *testing.T) {
			key := registry.GetProviderKeyFromOption("Invalid Option String")
			if key != "" {
				t.Errorf("Expected empty string for invalid option, got '%s'", key)
			}
		})
	})
}
