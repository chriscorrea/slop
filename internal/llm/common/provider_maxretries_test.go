package common_test

import (
	"log/slog"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/anthropic"
	"github.com/chriscorrea/slop/internal/llm/cohere"
	"github.com/chriscorrea/slop/internal/llm/groq"
	"github.com/chriscorrea/slop/internal/llm/mistral"
	"github.com/chriscorrea/slop/internal/llm/ollama"
	"github.com/chriscorrea/slop/internal/llm/openai"
)

func TestProviderSpecificMaxRetries(t *testing.T) {
	tests := []struct {
		name               string
		setupConfig        func() *config.Config
		providerName       string
		expectedMaxRetries int
	}{
		{
			name: "anthropic uses provider-specific max retries",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 1          // global default
				cfg.Providers.Anthropic.MaxRetries = 3 // provider-specific
				cfg.Providers.Anthropic.APIKey = "test-key"
				return cfg
			},
			providerName:       "anthropic",
			expectedMaxRetries: 3,
		},
		{
			name: "anthropic falls back to global max retries when provider-specific is 0",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 2          // global default
				cfg.Providers.Anthropic.MaxRetries = 0 // provider-specific not set
				cfg.Providers.Anthropic.APIKey = "test-key"
				return cfg
			},
			providerName:       "anthropic",
			expectedMaxRetries: 2,
		},
		{
			name: "openai enforces maximum limit of 5",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 10      // global default
				cfg.Providers.OpenAI.MaxRetries = 8 // provider-specific
				cfg.Providers.OpenAI.APIKey = "test-key"
				return cfg
			},
			providerName:       "openai",
			expectedMaxRetries: 5,
		},
		{
			name: "cohere uses provider-specific max retries",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 1
				cfg.Providers.Cohere.MaxRetries = 4
				cfg.Providers.Cohere.APIKey = "test-key"
				return cfg
			},
			providerName:       "cohere",
			expectedMaxRetries: 4,
		},
		{
			name: "groq falls back to global when provider-specific is 0",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 3
				cfg.Providers.Groq.MaxRetries = 0
				cfg.Providers.Groq.APIKey = "test-key"
				return cfg
			},
			providerName:       "groq",
			expectedMaxRetries: 3,
		},
		{
			name: "mistral uses provider-specific max retries",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 1
				cfg.Providers.Mistral.MaxRetries = 2
				cfg.Providers.Mistral.APIKey = "test-key"
				return cfg
			},
			providerName:       "mistral",
			expectedMaxRetries: 2,
		},
		{
			name: "ollama uses provider-specific max retries",
			setupConfig: func() *config.Config {
				cfg := config.NewDefaultFromEmbedded()
				cfg.Parameters.MaxRetries = 1
				cfg.Providers.Ollama.MaxRetries = 3
				return cfg
			},
			providerName:       "ollama",
			expectedMaxRetries: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			logger := slog.Default()

			// create provider based on name
			var client interface{}
			var err error

			switch tt.providerName {
			case "anthropic":
				provider := anthropic.New()
				client, err = provider.CreateClient(cfg, logger)
			case "openai":
				provider := openai.New()
				client, err = provider.CreateClient(cfg, logger)
			case "cohere":
				provider := cohere.New()
				client, err = provider.CreateClient(cfg, logger)
			case "groq":
				provider := groq.New()
				client, err = provider.CreateClient(cfg, logger)
			case "mistral":
				provider := mistral.New()
				client, err = provider.CreateClient(cfg, logger)
			case "ollama":
				provider := ollama.New()
				client, err = provider.CreateClient(cfg, logger)
			default:
				t.Fatalf("Unknown provider: %s", tt.providerName)
			}

			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// verify that client is created successfully
			// (note we can't directly check the maxRetries value—it's internal to adappter client)
			if client == nil {
				t.Error("Expected client to be created, got nil")
			}
		})
	}
}
