package app

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"slop/internal/config"
	"slop/internal/llm/common"
	"slop/internal/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLM implements the common.LLM interface for testing
type MockLLM struct {
	mock.Mock
}

// mocks the Generate method
func (m *MockLLM) Generate(ctx context.Context, messages []common.Message, modelName string, options ...interface{}) (string, error) {
	args := m.Called(ctx, messages, modelName, options)
	return args.String(0), args.Error(1)
}

// MockProvider implements registry.Provider for testing
type MockProvider struct {
	mockLLM *MockLLM
}

func (m *MockProvider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	return m.mockLLM, nil
}

func (m *MockProvider) BuildOptions(cfg *config.Config) []interface{} {
	return []interface{}{}
}

func (m *MockProvider) RequiresAPIKey() bool {
	return false
}

func (m *MockProvider) ProviderName() string {
	return "mock-provider"
}

func (m *MockProvider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	return nil, nil
}

func (m *MockProvider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	return "", nil, nil
}

func (m *MockProvider) HandleError(statusCode int, body []byte) error {
	return nil
}

func (m *MockProvider) CustomizeRequest(req *http.Request) error {
	return nil
}

func (m *MockProvider) HandleConnectionError(err error) error {
	return err
}

// setupMockRegistry is a helper function that sets up a mock provider
// this returns a cleanup function to restore the original registry state
func setupMockRegistry(mockProvider *MockProvider) func() {
	originalProviders := registry.AllProviders
	registry.AllProviders = map[string]common.Provider{
		"test-provider": mockProvider,
	}
	return func() {
		registry.AllProviders = originalProviders
	}
}

func TestApp_Run_Success(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify we have both system and user messages
			return len(messages) == 2 &&
				messages[0].Role == "system" &&
				messages[1].Role == "user" &&
				messages[1].Content == "test input"
		}),
		"test-model",
		mock.Anything).Return("mocked response", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
			Temperature:  0.7,
			MaxTokens:    100,
		},
		Format: config.Format{},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, err := app.Run(ctx, []string{"test input"}, []string{}, "", "test-provider", "test-model")

	assert.NoError(t, err)
	assert.Equal(t, "mocked response", result)

	mockLLM.AssertExpectations(t)
}

func TestApp_Run_CreateProvider_Error(t *testing.T) {
	// creste test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	// execute Run with invalid provider
	ctx := context.Background()
	result, err := app.Run(ctx, []string{"test input"}, []string{}, "", "nonexistent", "test-model")

	// assert results
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to create provider")
	assert.Contains(t, err.Error(), "unsupported provider 'nonexistent'")
}

func TestApp_Run_Generate_Error(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	expectedError := assert.AnError
	mockLLM.On("Generate",
		context.Background(),
		mock.Anything,
		"test-model",
		mock.Anything).Return("", expectedError)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	// execute Run
	ctx := context.Background()
	result, err := app.Run(ctx, []string{"test input"}, []string{}, "", "test-provider", "test-model")

	// assert results
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to generate response")
	assert.ErrorIs(t, err, expectedError)

	// verify mock expectations
	mockLLM.AssertExpectations(t)
}

func TestApp_Run_EnhanceSystemPrompt(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// Verify system prompt contains both base prompt and JSON instruction
			if len(messages) < 1 || messages[0].Role != "system" {
				return false
			}
			systemContent := messages[0].Content
			return assert.Contains(t, systemContent, "base prompt") &&
				assert.Contains(t, systemContent, "You must format your entire response as a single, valid JSON object")
		}),
		"test-model",
		mock.Anything).Return("{'result': 'json response'}", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config with JSON format
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "base prompt",
			Temperature:  0.7,
		},
		Format: config.Format{
			JSON: true,
		},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	// execute Run
	ctx := context.Background()
	result, err := app.Run(ctx, []string{"test input"}, []string{}, "", "test-provider", "test-model")

	// assert results
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	mockLLM.AssertExpectations(t)
}

func TestApp_Run_NoInput_Error(t *testing.T) {
	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "",
		},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	// execute Run with no input
	ctx := context.Background()
	result, err := app.Run(ctx, []string{}, []string{}, "", "mock", "test-model")

	// assert results - should fail with no input!
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no input provided")
}

func TestApp_Run_NilConfig_Error(t *testing.T) {
	// create app with nil config
	app := NewApp(nil, slog.Default(), false)

	// execute Run
	ctx := context.Background()
	result, err := app.Run(ctx, []string{"test input"}, []string{}, "", "test-provider", "test-model")

	// assert results
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "configuration is nil")
}

func TestApp_Run_WithCommandContext(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// should have system message and user message with command
			return len(messages) == 2 &&
				messages[0].Role == "system" &&
				messages[1].Role == "user"
		}),
		"test-model",
		mock.Anything).Return("response with context", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	// execute Run with command context
	ctx := context.Background()
	result, err := app.Run(ctx, []string{"test input"}, []string{}, "command context", "test-provider", "test-model")

	// assert results
	assert.NoError(t, err)
	assert.Equal(t, "response with context", result)

	// verify mock expectations
	mockLLM.AssertExpectations(t)
}

func TestApp_Run_DifferentFormats(t *testing.T) {
	tests := []struct {
		name          string
		format        config.Format
		expectedInSys string
	}{
		{
			name:          "YAML Format",
			format:        config.Format{YAML: true},
			expectedInSys: "valid YAML",
		},
		{
			name:          "Markdown Format",
			format:        config.Format{MD: true},
			expectedInSys: "valid Markdown",
		},
		{
			name:          "JSON Format",
			format:        config.Format{JSON: true},
			expectedInSys: "JSON",
		},
		{
			name:          "XML Format",
			format:        config.Format{XML: true},
			expectedInSys: "valid XML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup mock LLM for this test
			mockLLM := &MockLLM{}
			mockLLM.On("Generate",
				context.Background(),
				mock.MatchedBy(func(messages []common.Message) bool {
					if len(messages) < 1 || messages[0].Role != "system" {
						return false
					}
					return assert.Contains(t, messages[0].Content, tt.expectedInSys)
				}),
				"test-model",
				mock.Anything).Return("formatted response", nil)

			// setup mock registry
			mockProvider := &MockProvider{mockLLM: mockLLM}
			defer setupMockRegistry(mockProvider)()

			// create test config with specific format
			cfg := &config.Config{
				Parameters: config.Parameters{
					SystemPrompt: "base prompt",
				},
				Format: tt.format,
			}

			// create app
			app := NewApp(cfg, slog.Default(), false)

			// execute Run
			ctx := context.Background()
			result, err := app.Run(ctx, []string{"test input"}, []string{}, "", "test-provider", "test-model")

			// assert results
			assert.NoError(t, err)
			assert.Equal(t, "formatted response", result)

			// verify mock expectations
			mockLLM.AssertExpectations(t)
		})
	}
}

func TestGetSpinner(t *testing.T) {
	tests := []struct {
		name             string
		providerName     string
		modelName        string
		expectedGlyphs   []string
		expectedSpeed    int
		expectedNumGlyph int // minimum number of glyphs expected
	}{
		{
			name:             "Ollama provider",
			providerName:     "ollama",
			modelName:        "llama2",
			expectedSpeed:    333,
			expectedNumGlyph: 6,
		},
		{
			name:             "Default fallback",
			providerName:     "unknown",
			modelName:        "unknown-model",
			expectedSpeed:    200,
			expectedNumGlyph: 14,
		},
		{
			name:             "Case insensitive Claude",
			providerName:     "ANTHROPIC",
			modelName:        "CLAUDE-SONNET",
			expectedSpeed:    500, // quick star patterns
			expectedNumGlyph: 5,
		},
		{
			name:             "Case insensitive OpenAI",
			providerName:     "OPENAI",
			modelName:        "GPT-4",
			expectedSpeed:    125, // slow dots pattern
			expectedNumGlyph: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glyphs, speed := getSpinner(tt.providerName, tt.modelName)

			// verify speed
			assert.Equal(t, tt.expectedSpeed, speed)

			// verify number of glyphs
			assert.Len(t, glyphs, tt.expectedNumGlyph)

		})
	}
}
