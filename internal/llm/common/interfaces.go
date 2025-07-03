package common

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/chriscorrea/slop/internal/config"
)

// LLM is the client interface; all provider clients must implement this
type LLM interface {
	Generate(ctx context.Context, messages []Message, modelName string, options ...interface{}) (string, error)
}

// Provider is the unified interface that every provider must implement
// this interface combines factory & adapter roles into a single contract
type Provider interface {
	CreateClient(cfg *config.Config, logger *slog.Logger) (LLM, error)
	BuildOptions(cfg *config.Config) []interface{}
	RequiresAPIKey() bool
	ProviderName() string
	BuildRequest(messages []Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error)
	ParseResponse(body []byte, logger *slog.Logger) (content string, usage *Usage, err error)
	HandleError(statusCode int, body []byte) error
	CustomizeRequest(req *http.Request) error
	HandleConnectionError(err error) error
}
