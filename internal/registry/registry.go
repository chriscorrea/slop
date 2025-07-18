package registry

import (
	"fmt"
	"log/slog"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/anthropic"
	"github.com/chriscorrea/slop/internal/llm/cohere"
	"github.com/chriscorrea/slop/internal/llm/common"
	"github.com/chriscorrea/slop/internal/llm/groq"
	"github.com/chriscorrea/slop/internal/llm/mistral"
	"github.com/chriscorrea/slop/internal/llm/mock"
	"github.com/chriscorrea/slop/internal/llm/ollama"
	"github.com/chriscorrea/slop/internal/llm/openai"
	"github.com/chriscorrea/slop/internal/llm/together"
)

// AllProviders contains registered LLM providers
// TODO this is manually updated for now
var AllProviders = map[string]common.Provider{
	"anthropic": anthropic.New(),
	"cohere":    cohere.New(),
	"groq":      groq.New(),
	"mistral":   mistral.New(),
	"mock":      mock.New(),
	"ollama":    ollama.New(),
	"openai":    openai.New(),
	"together":  together.New(),
}

// CreateProvider creates a provider instance using the central registry
// this will return an error if provider is not registered or creation fails
func CreateProvider(name string, cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	provider, exists := AllProviders[name]
	if !exists {
		return nil, fmt.Errorf("unsupported provider '%s'. Available providers: %s", name, getAvailableProviders())
	}

	return provider.CreateClient(cfg, logger)
}

// BuildProviderOptions builds provider-specific options using the central registry
// returns nil if the provider is not registered
func BuildProviderOptions(name string, cfg *config.Config) []interface{} {
	provider, exists := AllProviders[name]
	if !exists {
		return nil
	}

	return provider.BuildOptions(cfg)
}

// GetAvailableProviders returns a list of registered provider names
func GetAvailableProviders() []string {
	providers := make([]string, 0, len(AllProviders))
	for name := range AllProviders {
		providers = append(providers, name)
	}
	return providers
}

// getAvailableProviders returns comma-separated string of available providers
func getAvailableProviders() string {
	providers := GetAvailableProviders()
	if len(providers) == 0 {
		return "none"
	}

	result := ""
	for i, provider := range providers {
		if i > 0 {
			result += ", "
		}
		result += provider
	}
	return result
}

// IsProviderRegistered checks if provider is registered
func IsProviderRegistered(name string) bool {
	_, exists := AllProviders[name]
	return exists
}

// ProviderRequiresAPIKey checks if provider requires API key
// returns false if the provider is not registered
func ProviderRequiresAPIKey(name string) bool {
	provider, exists := AllProviders[name]
	if !exists {
		return false
	}

	return provider.RequiresAPIKey()
}
