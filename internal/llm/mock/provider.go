package mock

import (
	"log/slog"
	"net/http"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// Provider implements the unified registry.Provider interface for Mock
type Provider struct{}

var _ common.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new mock LLM client
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	return &Client{}, nil
}

// BuildOptions creates mock-specific generation options from configuration
func (p *Provider) BuildOptions(cfg *config.Config) []interface{} {
	// Mock provider doesn't need real options here
	return []interface{}{}
}

// RequiresAPIKey returns false - no API key required!
func (p *Provider) RequiresAPIKey() bool {
	return false
}

// ProviderName returns the name of this provider
func (p *Provider) ProviderName() string {
	return "mock"
}

// BuildRequest creates a mock request
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	return map[string]interface{}{
		"model":    modelName,
		"messages": messages,
	}, nil
}

// Note that these are not really used in practice;
// we implement these to satisfy the interface

// ParseResponse parses a mock response
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	return "Mock LLM response", &common.Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}, nil
}

// HandleError handles mock errors
func (p *Provider) HandleError(statusCode int, body []byte) error {
	return nil
}

// HandleConnectionError handles mock connection errors
func (p *Provider) HandleConnectionError(err error) error {
	return err
}

// CustomizeRequest customizes mock requests
func (p *Provider) CustomizeRequest(req *http.Request) error {
	return nil
}
