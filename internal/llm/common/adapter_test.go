package common

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chriscorrea/slop/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProvider implements registry.Provider interface for testing
type MockProvider struct {
	mock.Mock
}

// mock the CreateClient method
func (m *MockProvider) CreateClient(cfg *config.Config, logger *slog.Logger) (LLM, error) {
	args := m.Called(cfg, logger)
	return args.Get(0).(LLM), args.Error(1)
}

// mock the BuildOptions method
func (m *MockProvider) BuildOptions(cfg *config.Config) []interface{} {
	args := m.Called(cfg)
	return args.Get(0).([]interface{})
}

// mock the RequiresAPIKey method
func (m *MockProvider) RequiresAPIKey() bool {
	args := m.Called()
	return args.Bool(0)
}

// mock the ProviderName method
func (m *MockProvider) ProviderName() string {
	args := m.Called()
	return args.String(0)
}

// mock the BuildRequest method
func (m *MockProvider) BuildRequest(messages []Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	args := m.Called(messages, modelName, options, logger)
	return args.Get(0), args.Error(1)
}

// mock the ParseResponse method
func (m *MockProvider) ParseResponse(body []byte, logger *slog.Logger) (string, *Usage, error) {
	args := m.Called(body, logger)
	return args.String(0), args.Get(1).(*Usage), args.Error(2)
}

// mock the HandleError method
func (m *MockProvider) HandleError(statusCode int, body []byte) error {
	args := m.Called(statusCode, body)
	return args.Error(0)
}

// mock the CustomizeRequest method
func (m *MockProvider) CustomizeRequest(req *http.Request) error {
	args := m.Called(req)
	return args.Error(0)
}

// mock the HandleConnectionError method
func (m *MockProvider) HandleConnectionError(err error) error {
	args := m.Called(err)
	return args.Error(0)
}

func TestAdapterClient_Generate_Success(t *testing.T) {
	// setup mock HTTP server returning successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"content": "hello", "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// setup mock provider
	mockProvider := &MockProvider{}

	// configure mock expectations in call order
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	// ProviderName called multiple times
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)

	mockProvider.On("ParseResponse", mock.AnythingOfType("[]uint8"), mock.Anything).Return(
		"hello",
		&Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		nil)

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", server.URL)

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.NoError(t, err)
	assert.Equal(t, "hello", result)

	// verify all mock expectations were met
	mockProvider.AssertExpectations(t)

	// verify specific methods were called
	mockProvider.AssertCalled(t, "BuildRequest", messages, "test-model", mock.Anything, mock.Anything)
	mockProvider.AssertCalled(t, "ParseResponse", mock.AnythingOfType("[]uint8"), mock.Anything)
}

func TestAdapterClient_Generate_BuildRequest_Error(t *testing.T) {
	// setup  mock provider
	mockProvider := &MockProvider{}

	// config  BuildRequest to return an error
	expectedError := errors.New("build request failed")
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(nil, expectedError)

	// No other methods should be called when BuildRequest fails

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", "http://localhost:8080")

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, result)

	// verify mock expectations
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "BuildRequest", messages, "test-model", mock.Anything, mock.Anything)
}

func TestAdapterClient_Generate_HTTP_Error(t *testing.T) {
	// setup  mock HTTP server that returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"error": "internal server error"}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// setup  mock provider
	mockProvider := &MockProvider{}

	// config  mock expectations in call order
	expectedError := errors.New("provider-specific error message")
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	// ProviderName called multiple times
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)

	mockProvider.On("HandleError",
		http.StatusInternalServerError,
		[]byte(`{"error": "internal server error"}`)).Return(expectedError)

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", server.URL)

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, result)

	// verify mock expectations
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "HandleError", http.StatusInternalServerError, mock.AnythingOfType("[]uint8"))
}

func TestAdapterClient_Generate_ParseResponse_Error(t *testing.T) {
	// setup  mock HTTP server that returns malformed JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"malformed": json}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// setup  mock provider
	mockProvider := &MockProvider{}

	// config  mock expectations in call order
	expectedError := errors.New("failed to parse response")
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	// ProviderName called multiple times
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)

	mockProvider.On("ParseResponse", []byte(`{"malformed": json}`), mock.Anything).Return("", (*Usage)(nil), expectedError)

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", server.URL)

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, result)

	// confirm mock expectations
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "ParseResponse", []byte(`{"malformed": json}`), mock.Anything)
}

func TestAdapterClient_Generate_Connection_Error(t *testing.T) {
	// setup  mock provider
	mockProvider := &MockProvider{}

	// config  mock expectations in call order
	enhancedError := errors.New("enhanced connection error message")

	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	// ProviderName called in marshalRequest()
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	// CustomizeRequest is called before connection fails
	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)

	mockProvider.On("HandleConnectionError", mock.MatchedBy(func(err error) bool {
		return err != nil
	})).Return(enhancedError)

	// create AdapterClient with invalid URL to force connection error
	client := NewAdapterClient(mockProvider, "test-key", "http://invalid-host:9999")

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.Error(t, err)
	assert.Equal(t, enhancedError, err)
	assert.Empty(t, result)

	// verify mock expectations
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "HandleConnectionError", mock.AnythingOfType("*url.Error"))
}

func TestAdapterClient_Generate_CustomizeRequest_Error(t *testing.T) {
	// setup  mock provider
	mockProvider := &MockProvider{}

	// config  mock expectations in call order
	customizeError := errors.New("failed to customize request")
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	// ProviderName called in marshalRequest()
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(customizeError)

	// CustomizeRequest error gets wrapped and passed to HandleConnectionError
	mockProvider.On("HandleConnectionError", mock.MatchedBy(func(err error) bool {
		return err != nil && err.Error() == "failed to customize request"
	})).Return(customizeError)

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", "http://localhost:8080")

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to customize request")
	assert.Empty(t, result)

	// verify mock expectations
	mockProvider.AssertExpectations(t)
}

func TestAdapterClient_Generate_With_Options(t *testing.T) {
	// setup mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"content": "response with options"}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// setup mock provider
	mockProvider := &MockProvider{}

	// test options
	testOptions := []interface{}{"option1", "option2"}

	// configure mock expectations in call order
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		"option1",
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
		"options":  testOptions,
	}, nil)

	// ProviderName called multiple times (allow unlimited calls for now)
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)
	mockProvider.On("ParseResponse", mock.AnythingOfType("[]uint8"), mock.Anything).Return("response with options", (*Usage)(nil), nil)

	// create AdapterClient with logger to avoid nil pointer dereference
	logger := slog.Default()
	client := NewAdapterClient(mockProvider, "test-key", server.URL, WithLogger(logger))

	// execute Generate with options
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model", testOptions...)

	// assert results
	assert.NoError(t, err)
	assert.Equal(t, "response with options", result)

	// verify mock expectations
	mockProvider.AssertExpectations(t)
}

func TestAdapterClient_Generate_Empty_Messages(t *testing.T) {
	// setup mock provider
	mockProvider := &MockProvider{}

	// config mock expectations in call order
	mockProvider.On("BuildRequest",
		[]Message{},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{},
	}, nil)

	// ProviderName called in marshalRequest() (and possibly readResponseBody())
	mockProvider.On("ProviderName").Return("test-provider").Maybe()

	// CustomizeRequest will be called before HTTP request
	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)

	// connection will fail to localhost:8080, so HandleConnectionError will be called
	mockProvider.On("HandleConnectionError", mock.MatchedBy(func(err error) bool {
		return err != nil
	})).Return(errors.New("connection failed"))

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", "http://localhost:8080")

	// exec client.Generate with empty messages
	ctx := context.Background()
	result, err := client.Generate(ctx, []Message{}, "test-model")

	// should still call BuildRequest even with empty messages
	// the provider can decide how to handle empty messages
	mockProvider.AssertCalled(t, "BuildRequest", []Message{}, "test-model", mock.Anything, mock.Anything)

	// the result depends on the full execution, but BuildRequest was called
	_ = result
	_ = err
}

func TestAdapterClient_Generate_ContextCancellation(t *testing.T) {
	// setup mock HTTP server that respects context cancellation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if context is already cancelled
		select {
		case <-r.Context().Done():
			// context was cancelled, which is what we want to test!
			return
		default:
			// context not cancelled yet–send response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"content": "response"}`))
		}
	}))
	defer server.Close()

	// setup mock provider
	mockProvider := &MockProvider{}

	// configure mock expectations; BuildRequest should be called
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	mockProvider.On("ProviderName").Return("test-provider").Maybe()
	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)

	// setup HandleConnectionError to expect context cancellation
	mockProvider.On("HandleConnectionError", mock.MatchedBy(func(err error) bool {
		return err != nil && err == context.Canceled
	})).Return(context.Canceled)

	// create AdapterClient
	client := NewAdapterClient(mockProvider, "test-key", server.URL)

	// create context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// exec client.Generate with cancelled context
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results–should get context cancellation error
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Empty(t, result)

	// verify mock expectations
	mockProvider.AssertExpectations(t)
}

func TestAdapterClient_Generate_AuthorizationHeader(t *testing.T) {
	// track the captured request
	var capturedRequest *http.Request

	// setup mock HTTP server that captures the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r // Capture the request for inspection
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"content": "authorized response"}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// setup mock provider
	mockProvider := &MockProvider{}

	// configure mock expectations
	mockProvider.On("BuildRequest",
		[]Message{{Role: "user", Content: "test message"}},
		"test-model",
		mock.Anything,
		mock.Anything).Return(map[string]interface{}{
		"model":    "test-model",
		"messages": []Message{{Role: "user", Content: "test message"}},
	}, nil)

	mockProvider.On("ProviderName").Return("test-provider").Maybe()
	mockProvider.On("CustomizeRequest", mock.AnythingOfType("*http.Request")).Return(nil)
	mockProvider.On("ParseResponse", mock.AnythingOfType("[]uint8"), mock.Anything).Return("authorized response", (*Usage)(nil), nil)

	// create AdapterClient with specific API key
	testAPIKey := "test-api-key-12345"
	client := NewAdapterClient(mockProvider, testAPIKey, server.URL)

	// execute Generate
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test message"}}
	result, err := client.Generate(ctx, messages, "test-model")

	// assert results
	assert.NoError(t, err)
	assert.Equal(t, "authorized response", result)

	// verify Authorization header was set correctly
	assert.NotNil(t, capturedRequest, "Request should have been captured")
	authHeader := capturedRequest.Header.Get("Authorization")
	expectedAuthHeader := "Bearer " + testAPIKey
	assert.Equal(t, expectedAuthHeader, authHeader, "Authorization header should be set correctly")

	// verify other expected headers
	assert.Equal(t, "application/json", capturedRequest.Header.Get("Content-Type"))

	// verify mock expectations
	mockProvider.AssertExpectations(t)
}
