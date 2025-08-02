package app

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	slopContext "github.com/chriscorrea/slop/internal/context"
	slopIO "github.com/chriscorrea/slop/internal/io"
	"github.com/chriscorrea/slop/internal/llm/common"
	"github.com/chriscorrea/slop/internal/registry"

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

// createEmptyContextResult is a helper function to create an empty ContextResult for testing
func createEmptyContextResult() *slopContext.ContextResult {
	return &slopContext.ContextResult{
		AllContextFiles:     []string{},
		CLIContextFiles:     []string{},
		CmdContextFiles:     []string{},
		ContextFileContents: []slopContext.ContextFile{},
	}
}

// createContextResultWithFiles creates a ContextResult with specified context files
func createContextResultWithFiles(contextFiles []slopContext.ContextFile) *slopContext.ContextResult {
	return &slopContext.ContextResult{
		AllContextFiles:     []string{},
		CLIContextFiles:     []string{},
		CmdContextFiles:     []string{},
		ContextFileContents: contextFiles,
	}
}

func TestApp_SyntheticMessageHistory_BasicScenario(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify we have system message + single user message (CLI args only)
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
		},
		Format: config.Format{},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"test input"}, createEmptyContextResult(), "", "test-provider", "test-model", "", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "mocked response", result)
	mockLLM.AssertExpectations(t)
}

func TestApp_SyntheticMessageHistory_ContextFiles(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify synthetic message history:
			// system + context file 1 + context file 2 + CLI args = 4 messages
			if len(messages) != 4 {
				return false
			}

			// verify system message
			if messages[0].Role != "system" {
				return false
			}

			// verify context file messages with proper formatting
			if messages[1].Role != "user" ||
				!assert.Contains(t, messages[1].Content, "File: /test/file1.txt") ||
				!assert.Contains(t, messages[1].Content, "content of file 1") {
				return false
			}

			if messages[2].Role != "user" ||
				!assert.Contains(t, messages[2].Content, "File: /test/file2.txt") ||
				!assert.Contains(t, messages[2].Content, "content of file 2") {
				return false
			}

			// verify CLI args as final user message
			if messages[3].Role != "user" || messages[3].Content != "analyze these files" {
				return false
			}

			return true
		}),
		"test-model",
		mock.Anything).Return("analysis response", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
		Format: config.Format{},
	}

	// create context result with multiple files
	contextFiles := []slopContext.ContextFile{
		{Path: "/test/file1.txt", Content: "content of file 1"},
		{Path: "/test/file2.txt", Content: "content of file 2"},
	}
	contextResult := createContextResultWithFiles(contextFiles)

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"analyze these files"}, contextResult, "", "test-provider", "test-model", "", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "analysis response", result)
	mockLLM.AssertExpectations(t)
}

func TestApp_SyntheticMessageHistory_ComplexScenario(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify synthetic message history:
			// system + ctx file + command ctx + arg = 4 messages
			if len(messages) != 4 {
				return false
			}

			// verify system message
			if messages[0].Role != "system" {
				return false
			}

			// verify context file message as first user message
			if messages[1].Role != "user" ||
				!assert.Contains(t, messages[1].Content, "File: /src/auth.go") ||
				!assert.Contains(t, messages[1].Content, "func authenticate(user string)") {
				return false
			}

			// verify command context as second user message
			if messages[2].Role != "user" || messages[2].Content != "You are reviewing windmill plans" {
				return false
			}

			// verify CLI args as final user message
			if messages[3].Role != "user" || messages[3].Content != "find security vulnerabilities" {
				return false
			}

			return true
		}),
		"test-model",
		mock.Anything).Return("security analysis", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a security expert",
		},
		Format: config.Format{},
	}

	// create context result with a single file
	contextFiles := []slopContext.ContextFile{
		{Path: "/src/auth.go", Content: "func authenticate(user string) { ... }"},
	}
	contextResult := createContextResultWithFiles(contextFiles)

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"find security vulnerabilities"}, contextResult, "You are reviewing windmill plans", "test-provider", "test-model", "", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "security analysis", result)
	mockLLM.AssertExpectations(t)
}

func TestApp_SyntheticMessageHistory_EmptyContextFiles(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify only system message + CLI args (empty context files should be skipped by manifest loading)
			return len(messages) == 2 &&
				messages[0].Role == "system" &&
				messages[1].Role == "user" &&
				messages[1].Content == "process files"
		}),
		"test-model",
		mock.Anything).Return("processed", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
		Format: config.Format{},
	}

	// create context result with empty files (empty content is already filtered out by manifest loading)
	contextFiles := []slopContext.ContextFile{
		// Note: manifest manager already filters out empty/whitespace files,
		// so ContextFiles here only contain non-empty content
	}
	contextResult := createContextResultWithFiles(contextFiles)

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"process files"}, contextResult, "", "test-provider", "test-model", "", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "processed", result)
	mockLLM.AssertExpectations(t)
}

func TestApp_SyntheticMessageHistory_ManyContextFiles(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify synthetic message history: system + 5 files + args = 7
			if len(messages) != 7 {
				return false
			}

			// verify system message
			if messages[0].Role != "system" {
				return false
			}

			// verify all context file messages
			for i := 1; i <= 5; i++ {
				if messages[i].Role != "user" ||
					!assert.Contains(t, messages[i].Content, "File: /test/file") ||
					!assert.Contains(t, messages[i].Content, "content") {
					return false
				}
			}

			// verify CLI arg as final user message
			if messages[6].Role != "user" || messages[6].Content != "summarize all files" {
				return false
			}

			return true
		}),
		"test-model",
		mock.Anything).Return("summary of all files", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
		Format: config.Format{},
	}

	// create context result with many files
	contextFiles := []slopContext.ContextFile{
		{Path: "/test/file1.txt", Content: "content 1"},
		{Path: "/test/file2.txt", Content: "content 2"},
		{Path: "/test/file3.txt", Content: "content 3"},
		{Path: "/test/file4.txt", Content: "content 4"},
		{Path: "/test/file5.txt", Content: "content 5"},
	}
	contextResult := createContextResultWithFiles(contextFiles)

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"summarize all files"}, contextResult, "", "test-provider", "test-model", "", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "summary of all files", result)
	mockLLM.AssertExpectations(t)
}

func TestBuildSyntheticMessageHistory_WithStdin(t *testing.T) {
	// create structured input with stdin content
	input := &slopIO.StructuredInput{
		ContextFiles: []slopContext.ContextFile{
			{Path: "/test/config.yaml", Content: "timeout: 30s\nretries: 3"},
		},
		StdinContent:   "piped data from stdin",
		CommandContext: "You are analyzing Squealer's configuration",
		CLIArgs:        "explain the configuration",
	}

	// build synthetic message history
	messages := buildSyntheticMessageHistory(input, nil, "")

	// verify correct order: context files, stdin, command context, CLI args
	assert.Len(t, messages, 4)

	// verify context file message (first)
	assert.Equal(t, "user", messages[0].Role)
	assert.Contains(t, messages[0].Content, "File: /test/config.yaml")
	assert.Contains(t, messages[0].Content, "timeout: 30s")

	// verify stdin content (second)
	assert.Equal(t, "user", messages[1].Role)
	assert.Equal(t, "piped data from stdin", messages[1].Content)

	// verify command context (third)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "You are analyzing Squealer's configuration", messages[2].Content)

	// verify CLI args (fourth/final)
	assert.Equal(t, "user", messages[3].Role)
	assert.Equal(t, "explain the configuration", messages[3].Content)
}

func TestBuildSyntheticMessageHistory_AllMessageTypes(t *testing.T) {
	// create structured input with all message types
	input := &slopIO.StructuredInput{
		ContextFiles: []slopContext.ContextFile{
			{Path: "/farm/windmill.go", Content: "package windmill\n\nfunc Build() { }"},
			{Path: "/farm/cowshed.go", Content: "package cowshed\n\nfunc Clean() { }"},
		},
		StdinContent:   "Boxer was working harder than ever",
		CommandContext: "You are reviewing Animal Farm construction plans",
		CLIArgs:        "analyze the farm infrastructure code",
	}

	// build synthetic message history
	messages := buildSyntheticMessageHistory(input, nil, "")

	// verify correct order and count: 2 context files + stdin + command context + CLI args = 5 messages
	assert.Len(t, messages, 5)

	// verify first context file message
	assert.Equal(t, "user", messages[0].Role)
	assert.Contains(t, messages[0].Content, "File: /farm/windmill.go")
	assert.Contains(t, messages[0].Content, "package windmill")

	// verify second context file message
	assert.Equal(t, "user", messages[1].Role)
	assert.Contains(t, messages[1].Content, "File: /farm/cowshed.go")
	assert.Contains(t, messages[1].Content, "package cowshed")

	// verify stdin content message
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "Boxer was working harder than ever", messages[2].Content)

	// verify command context message
	assert.Equal(t, "user", messages[3].Role)
	assert.Equal(t, "You are reviewing Animal Farm construction plans", messages[3].Content)

	// verify CLI args as final message
	assert.Equal(t, "user", messages[4].Role)
	assert.Equal(t, "analyze the farm infrastructure code", messages[4].Content)
}

// keep essential error and format tests
func TestApp_Run_CreateProvider_Error(t *testing.T) {
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
	}

	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"test input"}, createEmptyContextResult(), "", "nonexistent", "test-model", "", "")

	assert.Error(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to create provider")
	assert.Contains(t, err.Error(), "unsupported provider 'nonexistent'")
}

func TestApp_Run_Generate_Error(t *testing.T) {
	mockLLM := &MockLLM{}
	expectedError := assert.AnError
	mockLLM.On("Generate",
		context.Background(),
		mock.Anything,
		"test-model",
		mock.Anything).Return("", expectedError)

	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
	}

	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"test input"}, createEmptyContextResult(), "", "test-provider", "test-model", "", "")

	assert.Error(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to generate response")
	assert.ErrorIs(t, err, expectedError)

	mockLLM.AssertExpectations(t)
}

func TestApp_Run_NoInput_Error(t *testing.T) {
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "",
		},
	}

	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{}, createEmptyContextResult(), "", "mock", "test-model", "", "")

	assert.Error(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no input provided")
}

func TestApp_Run_NilConfig_Error(t *testing.T) {
	app := NewApp(nil, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"test input"}, createEmptyContextResult(), "", "test-provider", "test-model", "", "")

	assert.Error(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "configuration is nil")
}

func TestApp_Run_FormatEnhancement_JSON(t *testing.T) {
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify system prompt contains JSON formatting instruction
			if len(messages) < 1 || messages[0].Role != "system" {
				return false
			}
			systemContent := messages[0].Content
			return assert.Contains(t, systemContent, "base prompt") &&
				assert.Contains(t, systemContent, "valid JSON object")
		}),
		"test-model",
		mock.Anything).Return("{'result': 'json response'}", nil)

	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "base prompt",
		},
		Format: config.Format{
			JSON: true,
		},
	}

	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"test input"}, createEmptyContextResult(), "", "test-provider", "test-model", "", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.NotEmpty(t, result)
	mockLLM.AssertExpectations(t)
}

func TestGetSpinner(t *testing.T) {
	tests := []struct {
		name             string
		providerName     string
		modelName        string
		expectedSpeed    int
		expectedNumGlyph int
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
			expectedSpeed:    500,
			expectedNumGlyph: 5,
		},
		{
			name:             "Case insensitive OpenAI",
			providerName:     "OPENAI",
			modelName:        "GPT-4",
			expectedSpeed:    125,
			expectedNumGlyph: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glyphs, speed := getSpinner(tt.providerName, tt.modelName)

			assert.Equal(t, tt.expectedSpeed, speed)
			assert.Len(t, glyphs, tt.expectedNumGlyph)
		})
	}
}

func TestBuildSyntheticMessageHistory_WithMessageTemplate(t *testing.T) {
	tests := []struct {
		name            string
		messageTemplate string
		userInput       string
		expectedContent string
	}{
		{
			name:            "template with placeholder substitutes input at end",
			messageTemplate: "Please analyze: {input}",
			userInput:       "this code",
			expectedContent: "Please analyze: this code",
		},
		{
			name:            "multi-line template with placeholder substitutes input in middle",
			messageTemplate: "Please analyze these windmill plans:\n{input}\nProvide detailed feedback.",
			userInput:       "build it with care",
			expectedContent: "Please analyze these windmill plans:\nbuild it with care\nProvide detailed feedback.",
		},
		{
			name:            "template without placeholder simply prepends with line break",
			messageTemplate: "Code review:",
			userInput:       "function main() {}",
			expectedContent: "Code review:\nfunction main() {}",
		},
		{
			name:            "template without placeholder and empty input returns template",
			messageTemplate: "Generate a summary",
			userInput:       "",
			expectedContent: "Generate a summary",
		},
		{
			name:            "empty template returns user input",
			messageTemplate: "",
			userInput:       "hello world!",
			expectedContent: "hello world!",
		},
		{
			name:            "template with placeholder and empty input",
			messageTemplate: "Analyze this: {input}",
			userInput:       "",
			expectedContent: "Analyze this: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create structured input with just CLI args
			input := &slopIO.StructuredInput{
				CLIArgs: tt.userInput,
			}

			// build synthetic message history with template
			messages := buildSyntheticMessageHistory(input, nil, tt.messageTemplate)

			if tt.expectedContent == "" {
				// if expected content is empty, we should have no messages
				assert.Len(t, messages, 0)
			} else {
				// verify we have exactly one user message with the expected content
				assert.Len(t, messages, 1)
				assert.Equal(t, "user", messages[0].Role)
				assert.Equal(t, tt.expectedContent, messages[0].Content)
			}
		})
	}
}

func TestApp_Run_WithMessageTemplate(t *testing.T) {
	// setup mock LLM
	mockLLM := &MockLLM{}
	mockLLM.On("Generate",
		context.Background(),
		mock.MatchedBy(func(messages []common.Message) bool {
			// verify system message + processed user message
			return len(messages) == 2 &&
				messages[0].Role == "system" &&
				messages[1].Role == "user" &&
				messages[1].Content == "Please review: function main() {}"
		}),
		"test-model",
		mock.Anything).Return("Code review complete", nil)

	// setup mock registry
	mockProvider := &MockProvider{mockLLM: mockLLM}
	defer setupMockRegistry(mockProvider)()

	// create test config
	cfg := &config.Config{
		Parameters: config.Parameters{
			SystemPrompt: "You are a helpful assistant",
		},
		Format: config.Format{},
	}

	// create app
	app := NewApp(cfg, slog.Default(), false)

	ctx := context.Background()
	result, exitCode, err := app.Run(ctx, []string{"function main() {}"}, createEmptyContextResult(), "", "test-provider", "test-model", "Please review: {input}", "")

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "Code review complete", result)
	mockLLM.AssertExpectations(t)
}
