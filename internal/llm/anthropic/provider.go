// Package anthropic provides a client implementation for the Anthropic API.
//
// API Reference: https://docs.anthropic.com/en/api/messages
// Authentication: providers.anthropic.api_key or ANTHROPIC_API_KEY environment variable
//
// Example usage:
//   client := anthropic.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, anthropic.WithTemperature(0.7))
//

package anthropic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// Provider implements the unified registry
type Provider struct{}

// ensure Provider implements the common provider interface
var _ common.Provider = (*Provider)(nil)

// thinking budget defaults keyed on the cross-provider ThinkingLevel.
// medium targets moderate reasoning; high gives the model room to explore
const (
	thinkingBudgetMedium = 4000
	thinkingBudgetHigh   = 16000
)

// per-model max_tokens defaults. these keep headroom for extended thinking
// plus a generous completion on the larger Opus/Sonnet families while still
// matching smaller models' practical output sizes
const (
	maxTokensDefault       = 4096
	maxTokensSonnetFamily4 = 16384
	maxTokensOpusFamily4   = 32768
)

// New creates a new Anthropic provider instance
func New() *Provider {
	return &Provider{}
}

// CreateClient creates a new LLM client using the unified adapter pattern
func (p *Provider) CreateClient(cfg *config.Config, logger *slog.Logger) (common.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Providers.Anthropic.APIKey == "" {
		return nil, fmt.Errorf(`Anthropic API key is required.

You can set the API key using the environment variable ANTHROPIC_API_KEY or via slop config set anthropic-key=<your_api_key>
Get an API key from https://console.anthropic.com/settings/keys`)
	}

	// create client options
	var opts []common.ClientOption
	if cfg.Providers.Anthropic.BaseUrl != "" {
		opts = append(opts, common.WithBaseURL(cfg.Providers.Anthropic.BaseUrl))
	}
	if logger != nil {
		opts = append(opts, common.WithLogger(logger))
	}
	// use provider-specific MaxRetries, fall back to global if not set
	maxRetries := cfg.Providers.Anthropic.MaxRetries
	if maxRetries == 0 {
		maxRetries = cfg.Parameters.MaxRetries
	}
	if maxRetries > 5 {
		maxRetries = 5 // enforce maximum limit
	}
	if maxRetries > 0 {
		opts = append(opts, common.WithMaxRetries(maxRetries))
	}

	adapterClient := common.NewAdapterClient(p, cfg.Providers.Anthropic.APIKey, "https://api.anthropic.com/v1", opts...)
	return adapterClient, nil
}

// BuildOptions creates Anthropic-specific generation options from configuration
func (p *Provider) BuildOptions(cfg *config.Config) []interface{} {
	var functionalOpts []GenerateOption

	// handle system prompt from config
	if cfg.Parameters.SystemPrompt != "" {
		functionalOpts = append(functionalOpts, WithSystem(cfg.Parameters.SystemPrompt))
	}

	if cfg.Parameters.Temperature > 0 {
		functionalOpts = append(functionalOpts, WithTemperature(cfg.Parameters.Temperature))
	}
	if cfg.Parameters.MaxTokens > 0 {
		functionalOpts = append(functionalOpts, WithMaxTokens(cfg.Parameters.MaxTokens))
	}
	if cfg.Parameters.TopP > 0 {
		functionalOpts = append(functionalOpts, WithTopP(cfg.Parameters.TopP))
	}
	if len(cfg.Parameters.StopSequences) > 0 {
		functionalOpts = append(functionalOpts, WithStopSequences(cfg.Parameters.StopSequences))
	}
	if cfg.Format.JSON {
		functionalOpts = append(functionalOpts, WithJSONFormat())
	}

	// translate the cross-provider thinking level into Anthropic's native
	// extended-thinking block at request build time. silent no-op for
	// ThinkingOff and unknown values, so a user's config default survives
	// switching to a non-thinking model.
	if level, err := common.ParseThinkingLevel(cfg.Parameters.Thinking); err == nil && level != common.ThinkingOff {
		functionalOpts = append(functionalOpts, WithThinking(level))
	}

	// forward the pre-resolved response schema through common.WithSchema.
	// the adapter wires the schema onto Anthropic's output_config envelope
	// at request build time.
	if schema := strings.TrimSpace(cfg.Parameters.ResponseSchema); schema != "" {
		functionalOpts = append(functionalOpts, WithSchema("response", []byte(schema)))
	}

	return []interface{}{NewGenerateOptions(functionalOpts...)}
}

// RequiresAPIKey returns true; Anthropic requires an API key
func (p *Provider) RequiresAPIKey() bool {
	return true
}

// returns the name of this provider
func (p *Provider) ProviderName() string {
	return "anthropic"
}

// supportsThinking reports whether a model id accepts the extended-thinking
// block. conservative allowlist; unknown models silently skip the field so
// a stray --thinking flag never breaks a request
func supportsThinking(modelID string) bool {
	id := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(id, "claude-sonnet-4-"):
		return true
	case strings.HasPrefix(id, "claude-opus-4-"):
		return true
	case strings.HasPrefix(id, "claude-3-7-sonnet"):
		return true
	}
	return false
}

// thinkingBudget maps a cross-provider ThinkingLevel onto Anthropic's
// budget_tokens parameter. unknown levels get the medium budget so a
// caller with a stale level still gets a reasonable request
func thinkingBudget(level common.ThinkingLevel) int {
	switch level {
	case common.ThinkingHigh:
		return thinkingBudgetHigh
	case common.ThinkingMedium:
		return thinkingBudgetMedium
	default:
		return thinkingBudgetMedium
	}
}

// defaultMaxTokens returns a reasonable max_tokens value for a given model
// when the caller hasn't set one. claude-opus-4-* gets the largest budget,
// claude-sonnet-4-* a middle budget, everything else a modest default
func defaultMaxTokens(modelID string) int {
	id := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(id, "claude-opus-4-"):
		return maxTokensOpusFamily4
	case strings.HasPrefix(id, "claude-sonnet-4-"):
		return maxTokensSonnetFamily4
	}
	return maxTokensDefault
}

// anthropicVersionRE captures a trailing -<major>-<minor> suffix where the
// minor is one or two digits. that excludes 4.0-era date snapshots like
// claude-sonnet-4-20250514, whose minor component is an eight-digit date
var anthropicVersionRE = regexp.MustCompile(`-(\d+)-(\d{1,2})(?:-|$)`)

// parseAnthropicVersion extracts the major.minor from a Claude model id,
// or returns ok=false if no standard version suffix is detectable
func parseAnthropicVersion(modelID string) (major, minor int, ok bool) {
	m := anthropicVersionRE.FindStringSubmatch(strings.ToLower(modelID))
	if m == nil {
		return 0, 0, false
	}
	maj, err1 := strconv.Atoi(m[1])
	min, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return maj, min, true
}

// useAdaptiveThinking reports whether a model accepts Anthropic's adaptive
// thinking shape. 4.6 and later use adaptive+effort; 4.5 and earlier stay
// on the enabled+budget_tokens shape
func useAdaptiveThinking(modelID string) bool {
	major, minor, ok := parseAnthropicVersion(modelID)
	if !ok {
		return false
	}
	return major > 4 || (major == 4 && minor >= 6)
}

// supportsEffort reports whether a model accepts the output_config.effort
// parameter. per Anthropic's docs: Mythos Preview, Opus 4.5, Opus 4.6,
// Opus 4.7, and Sonnet 4.6. note that Opus 4.5 supports effort even
// though it uses manual (enabled+budget_tokens) thinking
func supportsEffort(modelID string) bool {
	id := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(id, "claude-opus-4-5"):
		return true
	case strings.HasPrefix(id, "claude-opus-4-6"):
		return true
	case strings.HasPrefix(id, "claude-opus-4-7"):
		return true
	case strings.HasPrefix(id, "claude-sonnet-4-6"):
		return true
	case strings.HasPrefix(id, "claude-mythos"):
		return true
	}
	return false
}

// supportsMaxEffort reports whether a model accepts effort="max". per
// Anthropic's docs, max is available on Opus 4.6, Opus 4.7, Sonnet 4.6,
// and the Mythos preview family. other adaptive models top out at "high"
func supportsMaxEffort(modelID string) bool {
	id := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(id, "claude-opus-4-6"):
		return true
	case strings.HasPrefix(id, "claude-opus-4-7"):
		return true
	case strings.HasPrefix(id, "claude-sonnet-4-6"):
		return true
	case strings.HasPrefix(id, "claude-mythos"):
		return true
	}
	return false
}

// effortForLevel maps slop's ThinkingLevel onto Anthropic's adaptive
// effort string. ThinkingHigh upgrades to "max" on models that support it,
// otherwise it falls back to "high"
func effortForLevel(level common.ThinkingLevel, maxOK bool) string {
	switch level {
	case common.ThinkingHigh:
		if maxOK {
			return "max"
		}
		return "high"
	case common.ThinkingMedium:
		return "medium"
	default:
		return "low"
	}
}

// BuildRequest creates an Anthropic-specific request from messages and options
func (p *Provider) BuildRequest(messages []common.Message, modelName string, options interface{}, logger *slog.Logger) (interface{}, error) {
	// convert options to Anthropic-specific options
	var config *GenerateOptions
	if options != nil {
		if anthropicOpts, ok := options.(*GenerateOptions); ok {
			config = anthropicOpts
		} else {
			config = &GenerateOptions{}
		}
	} else {
		config = &GenerateOptions{}
	}

	// log the API request using common utilities
	common.LogAPIRequest(logger, "Anthropic", modelName, messages, &config.GenerateOptions)

	// separate system messages from user/assistant messages
	var systemPrompt string
	var filteredMessages []common.Message

	for _, msg := range messages {
		if msg.Role == "system" {
			if systemPrompt == "" {
				systemPrompt = msg.Content
			} else {
				systemPrompt += "\n\n" + msg.Content
			}
		} else {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	// use system prompt from config if no system messages found
	if systemPrompt == "" && config.System != "" {
		systemPrompt = config.System
	}

	// create Anthropic-specific request payload. Anthropic requires
	// max_tokens, so seed a per-model default the caller can override
	requestBody := &MessagesRequest{
		Model:     modelName,
		Messages:  filteredMessages,
		MaxTokens: defaultMaxTokens(modelName),
		Stream:    common.BoolPtr(false), // disable streaming for now
	}

	// set system prompt if provided
	if systemPrompt != "" {
		requestBody.System = systemPrompt
	}

	// map common generation options to Anthropic's API format
	if config.Temperature != nil {
		requestBody.Temperature = config.Temperature
	}
	if config.MaxTokens != nil {
		requestBody.MaxTokens = *config.MaxTokens
	}
	if config.TopP != nil {
		requestBody.TopP = config.TopP
	}

	// map Anthropic-specific options
	if config.TopK != nil {
		requestBody.TopK = config.TopK
	}
	if len(config.StopSequences) > 0 {
		requestBody.StopSequences = config.StopSequences
	}

	// extended thinking block. the shape depends on the model:
	//   4.6+ — adaptive (no budget); the output_config.effort lever below
	//          steers depth, so we skip the thinking block only when the
	//          model doesn't support thinking at all
	//   4.5- — enabled+budget_tokens when the user asked for medium/high;
	//          off stays literal (no block at all)
	// unsupported models silently no-op so a default --thinking setting
	// survives switching to something like haiku
	if supportsThinking(modelName) {
		if useAdaptiveThinking(modelName) {
			requestBody.Thinking = &ThinkingConfig{Type: "adaptive"}
			// adaptive auto-manages tokens; no max_tokens bump needed
		} else if config.Thinking != common.ThinkingOff {
			budget := config.ThinkingBudget
			if budget <= 0 {
				budget = thinkingBudget(config.Thinking)
			}
			requestBody.Thinking = &ThinkingConfig{
				Type:         "enabled",
				BudgetTokens: budget,
			}

			// Anthropic requires max_tokens > budget_tokens on the legacy
			// shape. bump the ceiling when the caller's value is too tight
			// so --thinking high still has room to deliver an answer on top
			// of its reasoning tokens
			if requestBody.MaxTokens <= budget {
				adjusted := budget + maxTokensDefault
				if logger != nil {
					logger.Debug("adjusting max_tokens to satisfy thinking budget",
						"model", modelName,
						"budget_tokens", budget,
						"original_max_tokens", requestBody.MaxTokens,
						"adjusted_max_tokens", adjusted,
					)
				}
				requestBody.MaxTokens = adjusted
			}
		}
	}

	// effort lever on output_config. the allowlist is independent of
	// supportsThinking — Opus 4.5 uses manual thinking but still accepts
	// effort, and Mythos supports effort with adaptive-by-default
	if supportsEffort(modelName) {
		if requestBody.OutputConfig == nil {
			requestBody.OutputConfig = &OutputConfig{}
		}
		requestBody.OutputConfig.Effort = effortForLevel(config.Thinking, supportsMaxEffort(modelName))
	}

	// Anthropic only accepts temperature=1 whenever the thinking block is
	// present (adaptive or enabled), and models like Sonnet 4.6 reject
	// requests that set both temperature and top_p. force temperature to 1
	// and drop top_p so a --temperature/--top-p default survives switching
	// models that don't use thinking
	if requestBody.Thinking != nil {
		if requestBody.Temperature != nil && *requestBody.Temperature != 1 && logger != nil {
			logger.Debug("overriding temperature to 1 for extended thinking",
				"model", modelName,
				"original_temperature", *requestBody.Temperature,
			)
		}
		one := 1.0
		requestBody.Temperature = &one

		if requestBody.TopP != nil && logger != nil {
			logger.Debug("dropping top_p to satisfy anthropic's temperature/top_p exclusion",
				"model", modelName,
				"original_top_p", *requestBody.TopP,
			)
		}
		requestBody.TopP = nil
	}

	// structured output: wrap json_schema requests in Anthropic's
	// output_config envelope. non-schema formats (e.g. json_object) fall
	// through unchanged — Anthropic doesn't have a parallel for those.
	// merge onto the existing OutputConfig so an effort setting survives
	if rf := config.ResponseFormat; rf != nil && rf.Type == "json_schema" && len(rf.Schema) > 0 {
		if requestBody.OutputConfig == nil {
			requestBody.OutputConfig = &OutputConfig{}
		}
		requestBody.OutputConfig.Format = &OutputFormat{
			Type:   "json_schema",
			Name:   rf.Name,
			Schema: rf.Schema,
			Strict: rf.Strict,
		}
	}

	return requestBody, nil
}

// ParseResponse parses an Anthropic API response and extracts content and usage.
// thinking blocks are re-inlined as a <think>...</think> prefix on the content
// string so downstream format.ApplyThinkingFilter treats Anthropic's structured
// thinking identically to an inline-tag thinking stream
func (p *Provider) ParseResponse(body []byte, logger *slog.Logger) (string, *common.Usage, error) {
	// parse the response using Anthropic's Messages API format
	var anthropicResp MessagesResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		common.LogJSONUnmarshalError(logger, err, string(body))
		return "", nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	// extract content from the content array
	if len(anthropicResp.Content) == 0 {
		return "", nil, fmt.Errorf("no content in Anthropic response")
	}

	// walk the content array once, collecting text and thinking blocks
	// into separate buffers. Anthropic emits thinking blocks before the
	// text block they precede, so preserving order is not required — we
	// just concatenate each kind in stream order
	var textParts []string
	var thinkingParts []string
	for _, item := range anthropicResp.Content {
		switch item.Type {
		case "text":
			textParts = append(textParts, item.Text)
		case "thinking":
			if item.Thinking != "" {
				thinkingParts = append(thinkingParts, item.Thinking)
			}
		}
	}

	if len(textParts) == 0 {
		return "", nil, fmt.Errorf("no text content in Anthropic response")
	}

	content := strings.Join(textParts, "")

	// re-inline thinking as a <think> tag so the downstream filter treats
	// Anthropic structured thinking the same as any other provider's
	// inline-tag thinking stream
	if len(thinkingParts) > 0 {
		content = "<think>" + strings.Join(thinkingParts, "") + "</think>\n" + content
	}

	// convert Anthropic usage to common format
	var usage *common.Usage
	if anthropicResp.Usage.InputTokens > 0 || anthropicResp.Usage.OutputTokens > 0 {
		usage = &common.Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		}
	}

	// return content and usage information
	return content, usage, nil
}

// HandleError creates Anthropic-specific error messages from HTTP error responses
func (p *Provider) HandleError(statusCode int, body []byte) error {

	// without the body, we can sometimes provide specific, actionable error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf(`Anthropic API authentication failed.

Check your API key and ensure it is set correctly.
You can set the API key using the environment variable ANTHROPIC_API_KEY or via slop config set anthropic-key=<your_api_key>
Get an API key from https://console.anthropic.com/settings/keys`)

	case http.StatusTooManyRequests:
		return fmt.Errorf(`Anthropic API rate limit exceeded.

Please try again later or check your limits at https://console.anthropic.com/settings/limits`)
	}

	// attempt to parse Anthropic's error format
	var errorResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		// FALLBACK if the response was not the expected JSON format
		return fmt.Errorf("Anthropic API request failed with status %d: %s", statusCode, string(body))
	}

	// now we can return a specific error message
	if errorResp.Error.Message != "" {
		return fmt.Errorf("Anthropic API error: %s", errorResp.Error.Message)
	}

	// final catch-all if parsing succeeded but the message was empty
	return fmt.Errorf("an unknown API error occurred (status %d)", statusCode)
}

// HandleConnectionError handles connection failures - for cloud services, return original error
func (p *Provider) HandleConnectionError(err error) error {
	return err
}

// Anthropic uses /v1/messages endpoint and requires specific headers
func (p *Provider) CustomizeRequest(req *http.Request) error {
	if strings.HasSuffix(req.URL.Path, "/chat/completions") {
		// handle both "/chat/completions" and "/v1/chat/completions"
		if strings.HasSuffix(req.URL.Path, "/v1/chat/completions") {
			req.URL.Path = strings.Replace(req.URL.Path, "/v1/chat/completions", "/v1/messages", 1)
		} else {
			req.URL.Path = strings.Replace(req.URL.Path, "/chat/completions", "/v1/messages", 1)
		}
	}

	// Anthropic requires x-api-key header instead of Authorization Bearer
	// see: https://docs.anthropic.com/en/api/overview#authentication
	authHeader := req.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		req.Header.Del("Authorization")
		req.Header.Set("x-api-key", apiKey)
	}

	// Anthropic requires specific API version headers
	req.Header.Set("anthropic-version", "2023-06-01")

	return nil
}
