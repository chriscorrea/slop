package registry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"slop/internal/config"
	"slop/internal/llm/common"

	"github.com/stretchr/testify/assert"
)

// mockProvider implements the registry.Provider interface for testing
type mockProvider struct {
	name               string
	requiresAPIKey     bool
	shouldFailCreate   bool
	createClientCalled bool
}

var _ common.Provider = (*mockProvider)(nil)

// CreateClient creates a mock LLM client
func (m *mockProvider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	m.createClientCalled = true
	if m.shouldFailCreate {
		return nil, assert.AnError
	}
	return &mockLLMClient{}, nil
}

// BuildOptions returns sample options for testing
func (m *mockProvider) BuildOptions(cfg *config.Config) []interface{} {
	return []interface{}{"option1", "option2"}
}

// RequiresAPIKey returns the configured API key requirement
func (m *mockProvider) RequiresAPIKey() bool {
	return m.requiresAPIKey
}

// ProviderName returns the mock provider name
func (m *mockProvider) ProviderName() string {
	return m.name
}

// BuildRequest creates a mock request
func (m *mockProvider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	return map[string]interface{}{
		"model":    modelName,
		"messages": messages,
	}, nil
}

// ParseResponse parses a mock response
func (m *mockProvider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	return "mock response", &common.Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}, nil
}

// HandleError handles mock errors
func (m *mockProvider) HandleError(statusCode int, body []byte) error {
	return assert.AnError
}

// CustomizeRequest customizes mock requests
func (m *mockProvider) CustomizeRequest(req *http.Request) error {
	return nil
}

// HandleConnectionError handles mock connection errors
func (m *mockProvider) HandleConnectionError(err error) error {
	return err
}

// mockLLMClient implements the common.LLM interface for testing
type mockLLMClient struct{}

// Generate implements the common.LLM interface
func (m *mockLLMClient) Generate(ctx context.Context, messages []common.Message, modelName string, options ...interface{}) (string, error) {
	return "mock LLM response", nil
}

func TestAllProvidersInitialization(t *testing.T) {
	t.Run("AllProviders map is initialized", func(t *testing.T) {
		assert.NotNil(t, AllProviders)
		assert.NotEmpty(t, AllProviders)
	})

	t.Run("Contains expected provider keys", func(t *testing.T) {
		expectedProviders := []string{"cohere", "mistral", "ollama"}

		for _, providerName := range expectedProviders {
			assert.Contains(t, AllProviders, providerName, "AllProviders should contain %s", providerName)
			assert.NotNil(t, AllProviders[providerName], "Provider %s should not be nil", providerName)
		}
	})

	t.Run("All providers implement the Provider interface", func(t *testing.T) {
		for name, provider := range AllProviders {
			assert.Implements(t, (*common.Provider)(nil), provider, "Provider %s should implement the Provider interface", name)
		}
	})
}

func TestCreateProvider(t *testing.T) {
	// Save original providers and restore after test
	originalProviders := AllProviders
	defer func() { AllProviders = originalProviders }()

	tests := []struct {
		name            string
		providerName    string
		setupMock       func()
		expectError     bool
		expectNilClient bool
	}{
		{
			name:         "Success - Valid provider",
			providerName: "test-provider",
			setupMock: func() {
				AllProviders["test-provider"] = &mockProvider{
					name:           "test-provider",
					requiresAPIKey: true,
				}
			},
			expectError:     false,
			expectNilClient: false,
		},
		{
			name:         "Success - Mock cohere provider",
			providerName: "cohere",
			setupMock: func() {
				// replace real cohere provider with mock for testing
				AllProviders["cohere"] = &mockProvider{
					name:           "cohere",
					requiresAPIKey: true, // keep the same API key requirement
				}
			},
			expectError:     false, // mock should succeed
			expectNilClient: false,
		},
		{
			name:            "Failure - Unknown provider",
			providerName:    "unknown-provider",
			setupMock:       func() {}, // no setup needed
			expectError:     true,
			expectNilClient: true,
		},
		{
			name:         "Failure - Provider creation fails",
			providerName: "failing-provider",
			setupMock: func() {
				AllProviders["failing-provider"] = &mockProvider{
					name:             "failing-provider",
					shouldFailCreate: true,
				}
			},
			expectError:     true,
			expectNilClient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup mock if needed
			tt.setupMock()

			// create test config
			cfg := &config.Config{
				Parameters: config.Parameters{
					Temperature: 0.7,
				},
			}
			logger := slog.Default()

			// call function under test
			client, err := CreateProvider(tt.providerName, cfg, logger)

			// validate results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNilClient {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
			}
		})
	}
}

// TestCreateProvider_RegistryLookup is a pure unit test that verifies the registry
// correctly looks up providers from AllProviders map and calls CreateClient
func TestCreateProvider_RegistryLookup(t *testing.T) {
	// save original providers and restore after test
	originalProviders := AllProviders
	defer func() { AllProviders = originalProviders }()

	// craete a fresh providers map for isolation
	AllProviders = make(map[string]common.Provider)

	// create a mock provider that tracks if CreateClient was called
	mockProv := &mockProvider{
		name:               "test-mock",
		requiresAPIKey:     false,
		shouldFailCreate:   false,
		createClientCalled: false,
	}

	// add mock to registry
	AllProviders["test-mock"] = mockProv

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			Temperature: 0.7,
		},
	}
	logger := slog.Default()

	// call function under test
	client, err := CreateProvider("test-mock", cfg, logger)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.True(t, mockProv.createClientCalled, "CreateClient should have been called on the provider")
	assert.IsType(t, &mockLLMClient{}, client)
}

func TestBuildProviderOptions(t *testing.T) {
	// save original providers and restore after test
	originalProviders := AllProviders
	defer func() { AllProviders = originalProviders }()

	tests := []struct {
		name         string
		providerName string
		setupMock    func()
		expectNil    bool
		expectedLen  int
	}{
		{
			name:         "Success - Valid provider",
			providerName: "test-provider",
			setupMock: func() {
				AllProviders["test-provider"] = &mockProvider{
					name: "test-provider",
				}
			},
			expectNil:   false,
			expectedLen: 2, // mockProvider returns 2 options
		},
		{
			name:         "Success - Mock mistral provider",
			providerName: "mistral",
			setupMock: func() {
				// replace real mistral provider with mock for testing
				AllProviders["mistral"] = &mockProvider{
					name:           "mistral",
					requiresAPIKey: true,
				}
			},
			expectNil:   false,
			expectedLen: 2, // mockProvider returns 2 options
		},
		{
			name:         "Failure - Unknown provider",
			providerName: "unknown-provider",
			setupMock:    func() {}, // no setup needed
			expectNil:    true,
			expectedLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup mock if needed
			tt.setupMock()

			// create test config
			cfg := &config.Config{
				Parameters: config.Parameters{
					Temperature: 0.7,
					MaxTokens:   100,
				},
				Format: config.Format{
					JSON: true,
				},
			}

			// call function under test
			options := BuildProviderOptions(tt.providerName, cfg)

			// validate results
			if tt.expectNil {
				assert.Nil(t, options)
			} else {
				assert.NotNil(t, options)
				if tt.expectedLen > 0 {
					assert.Len(t, options, tt.expectedLen)
				}
			}
		})
	}
}

func TestGetAvailableProviders(t *testing.T) {
	providers := GetAvailableProviders()

	assert.NotNil(t, providers)
	assert.NotEmpty(t, providers)

	// check that expected providers are present
	expectedProviders := []string{"cohere", "mistral", "ollama"}
	for _, expected := range expectedProviders {
		assert.Contains(t, providers, expected)
	}
}

func TestIsProviderRegistered(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		expected     bool
	}{
		{
			name:         "Registered provider - cohere",
			providerName: "cohere",
			expected:     true,
		},
		{
			name:         "Registered provider - mistral",
			providerName: "mistral",
			expected:     true,
		},
		{
			name:         "Registered provider - ollama",
			providerName: "ollama",
			expected:     true,
		},
		{
			name:         "Unregistered provider - entropic",
			providerName: "entropic", // fake provider
			expected:     false,
		},
		{
			name:         "Empty provider name",
			providerName: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProviderRegistered(tt.providerName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		expected     bool
	}{
		{
			name:         "Cohere requires API key",
			providerName: "cohere",
			expected:     true,
		},
		{
			name:         "Mistral requires API key",
			providerName: "mistral",
			expected:     true,
		},
		{
			name:         "Ollama does not require API key",
			providerName: "ollama",
			expected:     false,
		},
		{
			name:         "Unknown provider defaults to false",
			providerName: "entropic", // fake provider
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProviderRequiresAPIKey(tt.providerName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMockProvider_Interface(t *testing.T) {
	// test that our mock properly implements the interface
	mock := &mockProvider{
		name:           "test-mock",
		requiresAPIKey: true,
	}

	cfg := &config.Config{}
	logger := slog.Default()

	t.Run("CreateClient", func(t *testing.T) {
		client, err := mock.CreateClient(cfg, logger)
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.IsType(t, &mockLLMClient{}, client)
	})

	t.Run("BuildOptions", func(t *testing.T) {
		options := mock.BuildOptions(cfg)
		assert.NotNil(t, options)
		assert.Len(t, options, 2)
		assert.Equal(t, "option1", options[0])
		assert.Equal(t, "option2", options[1])
	})

	t.Run("RequiresAPIKey", func(t *testing.T) {
		result := mock.RequiresAPIKey()
		assert.True(t, result)
	})

	t.Run("ProviderName", func(t *testing.T) {
		name := mock.ProviderName()
		assert.Equal(t, "test-mock", name)
	})

	t.Run("BuildRequest", func(t *testing.T) {
		messages := []common.Message{{Role: "user", Content: "test"}}
		request, err := mock.BuildRequest(messages, "test-model", nil, slog.Default())
		assert.NoError(t, err)
		assert.NotNil(t, request)
	})

	t.Run("ParseResponse", func(t *testing.T) {
		content, usage, err := mock.ParseResponse([]byte("test response"), slog.Default())
		assert.NoError(t, err)
		assert.Equal(t, "mock response", content)
		assert.NotNil(t, usage)
		assert.Equal(t, 10, usage.PromptTokens)
		assert.Equal(t, 20, usage.CompletionTokens)
		assert.Equal(t, 30, usage.TotalTokens)
	})
}

// TestConcurrentRegistryAccess tests concurrent access to registry functions
// to detect race conditions in the global AllProviders map access
func TestConcurrentRegistryAccess(t *testing.T) {
	t.Parallel()

	const (
		numGoroutines          = 20
		iterationsPerGoroutine = 100
	)

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	// Launch multiple goroutines to access registry functions concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				// Test GetAvailableProviders (iterates over map)
				providers := GetAvailableProviders()
				if len(providers) == 0 {
					errChan <- fmt.Errorf("goroutine %d iteration %d: GetAvailableProviders returned empty result", goroutineID, j)
					return
				}

				// Test IsProviderRegistered with known provider
				isRegistered := IsProviderRegistered("cohere")
				if !isRegistered {
					errChan <- fmt.Errorf("goroutine %d iteration %d: cohere should be registered", goroutineID, j)
					return
				}

				// Test IsProviderRegistered with unknown provider
				unknownRegistered := IsProviderRegistered("unknown-provider-xyz")
				if unknownRegistered {
					errChan <- fmt.Errorf("goroutine %d iteration %d: unknown provider should not be registered", goroutineID, j)
					return
				}

				// Test ProviderRequiresAPIKey with different providers
				cohereRequiresKey := ProviderRequiresAPIKey("cohere")
				if !cohereRequiresKey {
					errChan <- fmt.Errorf("goroutine %d iteration %d: cohere should require API key", goroutineID, j)
					return
				}

				ollamaRequiresKey := ProviderRequiresAPIKey("ollama")
				if ollamaRequiresKey {
					errChan <- fmt.Errorf("goroutine %d iteration %d: ollama should not require API key", goroutineID, j)
					return
				}

				// Test ProviderRequiresAPIKey with unknown provider
				unknownRequiresKey := ProviderRequiresAPIKey("unknown-provider-xyz")
				if unknownRequiresKey {
					errChan <- fmt.Errorf("goroutine %d iteration %d: unknown provider should not require API key", goroutineID, j)
					return
				}

				// Verify consistency of GetAvailableProviders results
				providers2 := GetAvailableProviders()
				if len(providers) != len(providers2) {
					errChan <- fmt.Errorf("goroutine %d iteration %d: GetAvailableProviders returned inconsistent length", goroutineID, j)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors from goroutines
	for err := range errChan {
		t.Error(err)
	}
}
