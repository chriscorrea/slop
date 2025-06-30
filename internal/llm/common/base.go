package common

import (
	"log/slog"
	"net/http"
	"time"
)

// BaseClient contains common client configuration shared across all LLM providers
type BaseClient struct {
	APIKey     string
	HTTPClient *http.Client
	BaseURL    string
	Logger     *slog.Logger
	MaxRetries int
}

// ClientOption configures a BaseClient using the functional options pattern
type ClientOption func(*BaseClient)

// WithLogger sets the logger for any client
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *BaseClient) {
		c.Logger = logger
	}
}

// WithMaxRetries sets maximum retry attempts for any client
func WithMaxRetries(retries int) ClientOption {
	return func(c *BaseClient) {
		c.MaxRetries = retries
	}
}

// WithHTTPClient sets the HTTP client for any client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *BaseClient) {
		c.HTTPClient = client
	}
}

// WithBaseURL sets the base URL for any client
func WithBaseURL(url string) ClientOption {
	return func(c *BaseClient) {
		c.BaseURL = url
	}
}

// NewBaseClient creates a base client with sensible defaults
func NewBaseClient(apiKey, defaultBaseURL string, opts ...ClientOption) *BaseClient {
	c := &BaseClient{
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		BaseURL:    defaultBaseURL,
		MaxRetries: 2,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
