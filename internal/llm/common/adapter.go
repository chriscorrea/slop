package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ProviderAdapter defines the interface that each LLM provider MUST implement
// unified client handles common logic (HTTP, retries, logging)
// while delegating details thta are provider-specific to the adapter
type ProviderAdapter interface {
	// ProviderName returns the name of the provider (e.g., "mistral", "cohere")
	ProviderName() string

	// RequiresAPIKey returns true if this provider requires an API key for authentication
	RequiresAPIKey() bool

	// BuildRequest creates a provider-specific request payload from messages and options
	// options parameter contains provider-specific generation options
	// returns the request payload that will be JSON marshaled and sent via HTTP
	BuildRequest(messages []Message, modelName string, options interface{}) (interface{}, error)

	// ParseResponse parses a provider-specific HTTP response body into content and usage stats
	// returns the generated content, token usage information, and any parsing errors
	ParseResponse(body []byte) (content string, usage *Usage, err error)

	// HandleError creates provider-specific error messages from HTTP error responses
	// allows each provider to return helpful, actionable error messages
	HandleError(statusCode int, body []byte) error

	// HandleConnectionError creates provider-specific error messages for connection failures
	// allows providers to give helpful guidance when the service is unreachable
	// returns the original error if no special handling is needed
	HandleConnectionError(err error) error

	// CustomizeRequest allows providers to add custom headers or modify the HTTP request
	// most providers can leave this empty–some could require special headers, etc
	CustomizeRequest(req *http.Request) error
}

// AdapterClient is a unified client that works with any Provider
// handles all common operations (HTTP requests, retries, logging, validation)
// while delegating provider-specific logic to the adapter
type AdapterClient struct {
	*BaseClient
	adapter Provider
}

// ensure AdapterClient implements the LLM interface
var _ LLM = (*AdapterClient)(nil)

// NewAdapterClient creates a new unified client with the given adapter
func NewAdapterClient(adapter Provider, apiKey, baseURL string, opts ...ClientOption) *AdapterClient {
	base := NewBaseClient(apiKey, baseURL, opts...)
	return &AdapterClient{
		BaseClient: base,
		adapter:    adapter,
	}
}

// Generate implements the unified generation logic for all providers
// centralizes all common functionality while using the adapter for provider-specific details
// TODO? ...interface() pushes type checking to runtime; consider using a more structured approach
func (c *AdapterClient) Generate(ctx context.Context, messages []Message, modelName string, options ...interface{}) (string, error) {
	// combine all interface{} options into a single options object
	processedOptions, err := c.processOptions(options)
	if err != nil {
		return "", err
	}

	// use adapter to build provider-specific request
	request, err := c.adapter.BuildRequest(messages, modelName, processedOptions, c.Logger)
	if err != nil {
		return "", err
	}

	// HTTP request with common retry logic
	response, err := c.executeRequest(ctx, request)
	if err != nil {
		// allow adapter to provide better error messages for connection failures
		return "", c.adapter.HandleConnectionError(err)
	}
	defer response.Body.Close()

	// read the response body
	body, err := c.readResponseBody(response)
	if err != nil {
		return "", err
	}

	// handle errors; should be provider-specific error handling
	if response.StatusCode != http.StatusOK {
		return "", c.adapter.HandleError(response.StatusCode, body)
	}

	// yse adapter to parse provider-specific response
	content, usage, err := c.adapter.ParseResponse(body, c.Logger)
	if err != nil {
		return "", err
	}

	// validate JSON format if requested
	if err := c.validateJSONResponse(content, processedOptions); err != nil {
		return "", err
	}

	// log results
	c.logSuccess(content, usage)

	return content, nil
}

// processOptions handles the processed configuration object from providers
// Providers convert functional options into a single configuration object before calling AdapterClient
func (c *AdapterClient) processOptions(options []interface{}) (interface{}, error) {
	// providers should pass exactly one processed configuration object
	// multiple functional options are processed at the provider level, not here
	if len(options) > 1 {
		// indicates a provider implementation bug - they should consolidate options
		c.Logger.Warn("Multiple options passed to AdapterClient - only first will be used", "count", len(options))
	}

	if len(options) > 0 {
		return options[0], nil
	}
	return nil, nil
}

// executeRequest handles the common HTTP request execution with retry logic
func (c *AdapterClient) executeRequest(ctx context.Context, request interface{}) (*http.Response, error) {
	// marshal request to JSON
	jsonData, err := c.marshalRequest(request)
	if err != nil {
		return nil, err
	}

	// build URL - most providers use /chat/completions, but adapters can customize
	url := c.buildRequestURL()
	LogRequestExecution(c.Logger, url, c.MaxRetries)

	// create executor function for retry logic
	executor := func(ctx context.Context) (*http.Response, error) {
		// create fresh request for each attempt
		req, err := c.createHTTPRequest(ctx, url, jsonData)
		if err != nil {
			return nil, err
		}

		// allow adapter to customize request (headers, auth, etc.)
		if err := c.adapter.CustomizeRequest(req); err != nil {
			return nil, err
		}

		return c.HTTPClient.Do(req)
	}

	// execute with retry logic
	resp, err := ExecuteWithRetry(ctx, executor, c.MaxRetries, c.Logger)
	if err != nil {
		LogRequestFailure(c.Logger, err, c.MaxRetries)
		return nil, err
	}

	return resp, nil
}

// marshalRequest converts the request to JSON
func (c *AdapterClient) marshalRequest(request interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s request: %w", c.adapter.ProviderName(), err)
	}
	return jsonData, nil
}

// buildRequestURL constructs the API endpoint URL
func (c *AdapterClient) buildRequestURL() string {
	// most providers use the standard /chat/completions endpoint
	// adapters can override this in CustomizeRequest if needed
	return BuildChatCompletionsURL(c.BaseURL)
}

// createHTTPRequest creates the basic HTTP request with standard headers
func (c *AdapterClient) createHTTPRequest(ctx context.Context, url string, jsonData []byte) (*http.Request, error) {
	// Create request with standard JSON headers
	return CreateJSONRequest(ctx, url, c.APIKey, jsonData)
}

// readResponseBody reads and logs the HTTP response body
func (c *AdapterClient) readResponseBody(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s response body: %w", c.adapter.ProviderName(), err)
	}

	// log response details
	LogHTTPResponse(c.Logger, response.StatusCode, len(body))
	LogRawResponse(c.Logger, string(body), response.StatusCode)

	return body, nil
}

// logSuccess logs successful completion with token usage
func (c *AdapterClient) logSuccess(content string, usage *Usage) {
	if usage != nil {
		LogTokenUsage(c.Logger, "", *usage) // ID can be empty for unified logging
	}
	LogRequestCompletion(c.Logger, len(content))
}

// validateJSONResponse validates JSON format if structured output was requested
func (c *AdapterClient) validateJSONResponse(content string, options interface{}) error {
	// extract GenerateOptions from the provider-specific options
	// this isn't ideal, but necessary to access the common validation logic
	if options != nil {
		// use reflection or type assertion to extract GenerateOptions
		// for now, we'll try to get the embedded GenerateOptions field
		if genOpts := c.extractGenerateOptions(options); genOpts != nil {
			return ValidateJSONResponse(content, genOpts, c.Logger)
		}
	}
	return nil
}

// extractGenerateOptions tries to extract the common GenerateOptions from provider-specific options
func (c *AdapterClient) extractGenerateOptions(options interface{}) *GenerateOptions {
	if options == nil {
		return nil
	}

	// use type assertion to handle different provider option types
	// each provider options struct embeds common.GenerateOptions
	switch opts := options.(type) {
	case interface{ GetGenerateOptions() *GenerateOptions }:
		// if the options implement a getter method
		return opts.GetGenerateOptions()
	default:
		// for now, return nil - individual providers can override validation if needed
		// this is a limitation of the current architecture and could be improved
		return nil
	}
}
