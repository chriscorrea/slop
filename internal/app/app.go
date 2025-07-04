package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chriscorrea/slop/internal/config"
	slopContext "github.com/chriscorrea/slop/internal/context"
	"github.com/chriscorrea/slop/internal/format"
	slopIO "github.com/chriscorrea/slop/internal/io"
	"github.com/chriscorrea/slop/internal/llm/common"
	"github.com/chriscorrea/slop/internal/registry"
	"github.com/chriscorrea/slop/internal/verbose"

	"github.com/fatih/color"
)

// enhanceSystemPromptForFormat adds formatting instructions to the system prompt based on config
func enhanceSystemPromptForFormat(basePrompt string, format config.Format) string {
	var instruction string

	switch {
	case format.JSON:
		instruction = "You must format your entire response as a single, valid JSON object. Start your response with a single opening brace {"
	case format.JSONL:
		instruction = "You must format your response as newline-delimited JSON (JSONL). Each line must be a self-contained, valid JSON object. Do not use commas after each line; simply separate each JSON object with a newline."
	case format.YAML:
		instruction = "You must format your entire response as valid YAML. Do not include any text or formatting outside of the YAML structure."
	case format.MD:
		instruction = "You must format your entire response as valid Markdown. Use appropriate Markdown syntax including headers, lists, code blocks, and other formatting elements as needed."
	case format.XML:
		instruction = "You must format your entire response as valid XML. Use proper XML structure with opening and closing tags. Start your response with a single opening angle bracket <"
	}

	if instruction != "" {
		if basePrompt == "" {
			return instruction
		}
		return basePrompt + "\n\n" + instruction
	}

	return basePrompt
}

// cleanFormattedResponse removes text outside format boundaries
func cleanFormattedResponse(response string, cfg config.Format) string {
	return format.CleanResponse(response, cfg)
}

// App represents the main application and holds its dependencies
type App struct {
	cfg     *config.Config
	logger  *slog.Logger
	verbose bool
}

// NewApp creates a new App instance with the provided configuration, logger, and verbose setting
func NewApp(cfg *config.Config, logger *slog.Logger, verbose bool) *App {
	return &App{
		cfg:     cfg,
		logger:  logger,
		verbose: verbose,
	}
}

// getSpinnerChars returns spinner characters
// just for fun, these can vary based on provider/model
func getSpinner(providerName, modelName string) (glyphs []string, speed int) {
	searchText := strings.ToLower(providerName + " " + modelName)

	switch {
	case strings.Contains(searchText, "claude"):
		glyphs = []string{
			"✶",
			"✸",
			"✺",
			"✹",
			"✷",
		}
		speed = 500
	case strings.Contains(searchText, "open"):
		glyphs = []string{
			"⠋",
			"⠙",
			"⠚",
			"⠒",
			"⠂",
			"⠂",
			"⠒",
			"⠲",
			"⠴",
			"⠦",
			"⠖",
			"⠒",
			"⠐",
			"⠐",
			"⠒",
			"⠓",
			"⠋",
		}
		speed = 125
	case strings.Contains(searchText, "ollama"):
		glyphs = []string{
			"◜",
			"◠",
			"◝",
			"◞",
			"◡",
			"◟",
		}
		speed = 333
	default:
		glyphs = []string{"⠄",
			"⠆",
			"⠇",
			"⠋",
			"⠙",
			"⠸",
			"⠰",
			"⠠",
			"⠰",
			"⠸",
			"⠙",
			"⠋",
			"⠇",
			"⠆"}
		speed = 200

	}
	return
}

// Run executes the main application logic
func (a *App) Run(ctx context.Context, cliArgs []string, contextResult *slopContext.ContextResult, commandContext, providerName, modelName string) (string, error) {
	if a.cfg == nil {
		return "", fmt.Errorf("configuration is nil")
	}

	// read input using structured processing for synthetic message history
	var contextFiles []slopContext.ContextFile
	if contextResult != nil {
		contextFiles = contextResult.ContextFileContents
	}

	// calculate project context count for spinner display
	projectContextCount := len(contextFiles)

	structuredInput, err := slopIO.ReadInput(os.Stdin, cliArgs, contextFiles, commandContext)
	if err != nil {
		return "", fmt.Errorf("failed to read structured input: %w", err)
	}

	// create a provider using the registry
	provider, err := registry.CreateProvider(providerName, a.cfg, a.logger)
	if err != nil {
		return "", fmt.Errorf("failed to create provider: %w", err)
	}

	// create messages using either synthetic message history or traditional approach
	var messages []common.Message

	// apply format instructions regardless of native structured output support
	enhancedSystemPrompt := enhanceSystemPromptForFormat(a.cfg.Parameters.SystemPrompt, a.cfg.Format)

	if enhancedSystemPrompt != "" {
		messages = append(messages, common.Message{
			Role:    "system",
			Content: enhancedSystemPrompt,
		})
	}

	// build synthetic message history from structured input
	messages = append(messages, buildSyntheticMessageHistory(structuredInput)...)

	// if no messages created, return an error
	if len(messages) == 0 {
		return "", fmt.Errorf("no input provided")
	}

	// display verbose output if enabled
	if a.verbose {
		outputCfg := verbose.DefaultOutputConfig(os.Stderr)
		verbose.PrintLLMParameters(a.cfg, providerName, modelName, outputCfg)
	}

	// log request parameters when debug is enabled
	if a.logger != nil {
		// calculate total message content length for logging
		var totalContentLength int
		var userMessageCount int
		for _, msg := range messages {
			totalContentLength += len(msg.Content)
			if msg.Role == "user" {
				userMessageCount++
			}
		}

		a.logger.Info("Preparing LLM request",
			"provider", providerName,
			"model", modelName,
			"temperature", a.cfg.Parameters.Temperature,
			"max_tokens", a.cfg.Parameters.MaxTokens,
			"system_prompt_length", len(enhancedSystemPrompt),
			"total_content_length", totalContentLength,
			"user_message_count", userMessageCount,
			"synthetic_history", true)

		if enhancedSystemPrompt != "" {
			a.logger.Debug("System prompt", "content", enhancedSystemPrompt)
		}

		// log all user messages when debug is enabled
		for i, msg := range messages {
			if msg.Role == "user" {
				a.logger.Debug("User message", "index", i, "content_length", len(msg.Content), "content", msg.Content)
			}
		}
	}

	// build generation options from configuration using the registry
	opts := registry.BuildProviderOptions(providerName, a.cfg)

	// force color output for spinner, even in chained commands
	// (where TTY detection might cause color to be disabled)
	color.NoColor = false

	// spinner
	done := make(chan bool, 1) // buffered channel to prevent goroutine leaks
	go func() {
		defer func() {
			// always clear this line when the goroutine exits
			fmt.Fprintf(os.Stderr, "\r%s\r", "                                                                                ")
		}()

		// get spinner properties (informed by provider and model)
		spinGlyphs, spinSpeed := getSpinner(providerName, modelName)

		i := 0
		cyan := color.New(color.FgCyan).SprintFunc()

		for {
			select {
			case <-done:
				return
			case <-ctx.Done(): // handle context cancellation
				return
			case <-time.After(time.Duration(spinSpeed) * time.Millisecond):
				baseMessage := fmt.Sprintf("%s %s", spinGlyphs[i], modelName) // always display model name and glyph
				switch projectContextCount {
				case 0:
					baseMessage += " is generating..." // default
				case 1:
					if len(contextFiles) > 0 {
						fileName := filepath.Base(contextFiles[0].Path)
						baseMessage += fmt.Sprintf(" is generating (using %s)", fileName)
					} else {
						baseMessage += " is generating using 1 project context file..."
					}
				default:
					baseMessage += fmt.Sprintf(" is generating (using %d project context files)", projectContextCount)
				}
				// print the message
				fmt.Fprintf(os.Stderr, "\r%s", cyan(baseMessage))
				i = (i + 1) % len(spinGlyphs)
			}
		}
	}()

	// generate response using the provider with the specified model
	response, err := provider.Generate(ctx, messages, modelName, opts...)

	// stop the spinner
	done <- true
	// give a tiny moment for the goroutine to clean up
	time.Sleep(10 * time.Millisecond)

	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	// clean the response based on format requirements
	cleanedResponse := cleanFormattedResponse(response, a.cfg.Format)

	return cleanedResponse, nil
}

// buildSyntheticMessageHistory creates a sequence of user messages from structured input
func buildSyntheticMessageHistory(input *slopIO.StructuredInput) []common.Message {
	var messages []common.Message

	// 1: each context file becomes a separate user message
	for _, contextFile := range input.ContextFiles {
		if contextFile.Content != "" {
			// include file path information for better context?
			content := fmt.Sprintf("File: %s\n\n%s", contextFile.Path, contextFile.Content)
			messages = append(messages, common.Message{
				Role:    "user",
				Content: content,
			})
		}
	}

	// 2: stdin content becomes a user message (if present)
	if input.StdinContent != "" {
		messages = append(messages, common.Message{
			Role:    "user",
			Content: input.StdinContent,
		})
	}

	// 3: command context becomes a user message (if present)
	if input.CommandContext != "" {
		messages = append(messages, common.Message{
			Role:    "user",
			Content: input.CommandContext,
		})
	}

	// 4: CLI arg (user prompt) becomes the final/most recent message
	if input.CLIArgs != "" {
		messages = append(messages, common.Message{
			Role:    "user",
			Content: input.CLIArgs,
		})
	}

	return messages
}
