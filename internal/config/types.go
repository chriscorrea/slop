package config

// Config represents the complete configuration structure for slop
type Config struct {
	Parameters Parameters             `mapstructure:"parameters"`
	Models     Models                 `mapstructure:"models"`
	Providers  Providers              `mapstructure:"providers"`
	Commands   map[string]Command     `mapstructure:"commands"`
	ExitCodes  map[string]ExitCodeMap `mapstructure:"exit_codes"`
	Format     Format                 `mapstructure:"format"`
}

// Parameters contains default configuration values and model selection preferences
type Parameters struct {
	// model generation parameters
	Temperature   float64  `mapstructure:"temperature"`
	SystemPrompt  string   `mapstructure:"system_prompt"`
	MaxTokens     int      `mapstructure:"max_tokens"`
	TopP          float64  `mapstructure:"top_p"`
	StopSequences []string `mapstructure:"stop_sequences"`
	Stream        bool     `mapstructure:"stream"`
	Seed          *int     `mapstructure:"seed"`

	// default model selection preferences
	DefaultModelType string `mapstructure:"default_model_type"` // "fast" or "deep"
	DefaultLocation  string `mapstructure:"default_location"`   // "local" or "remote"

	// application behavior
	Timeout    int `mapstructure:"timeout"`
	MaxRetries int `mapstructure:"max_retries"`
}

// Format contains output formatting options
type Format struct {
	JSON  bool `mapstructure:"json"`
	JSONL bool `mapstructure:"jsonl"`
	YAML  bool `mapstructure:"yaml"`
	MD    bool `mapstructure:"md"`
	XML   bool `mapstructure:"xml"`
}

// ExitCodeRule defines a pattern and exit code
type ExitCodeRule struct {
	MatchType string `mapstructure:"match_type"` // "exact", "contains", "regex", "prefix", "suffix"
	Pattern   string `mapstructure:"pattern"`
	ExitCode  int    `mapstructure:"exit_code"`
}

// ExitCodeMap is a named set of rules for determining an exit code
type ExitCodeMap struct {
	Description string         `mapstructure:"description"`
	Rules       []ExitCodeRule `mapstructure:"rules"`
}

// Models contains model configuration for different categories
type Models struct {
	Remote Remote `mapstructure:"remote"`
	Local  Local  `mapstructure:"local"`
}

// Remote contains remote model definitions
type Remote struct {
	Fast Fast `mapstructure:"fast"`
	Deep Deep `mapstructure:"deep"`
}

// Local contains local model definitions
type Local struct {
	Fast Fast `mapstructure:"fast"`
	Deep Deep `mapstructure:"deep"`
}

// Fast represents a fast/lightweight model configuration
type Fast struct {
	Provider string `mapstructure:"provider"`
	Name     string `mapstructure:"name"`
}

// Deep represents a deep/reasoning model configuration
type Deep struct {
	Provider string `mapstructure:"provider"`
	Name     string `mapstructure:"name"`
}

// Providers contains configuration for different LLM providers
type Providers struct {
	Anthropic Anthropic `mapstructure:"anthropic"`
	OpenAI    OpenAI    `mapstructure:"openai"`
	Cohere    Cohere    `mapstructure:"cohere"`
	Ollama    Ollama    `mapstructure:"ollama"`
	Mistral   Mistral   `mapstructure:"mistral"`
	Groq      Groq      `mapstructure:"groq"`
	Together  Together  `mapstructure:"together"`
}

// BaseProvider contains common fields shared across all providers
type BaseProvider struct {
	APIKey     string `mapstructure:"api_key"`
	BaseUrl    string `mapstructure:"base_url"`
	APIVersion string `mapstructure:"api_version"`
	MaxRetries int    `mapstructure:"max_retries"`
}

type Anthropic struct {
	BaseProvider `mapstructure:",squash"`
}

type Cohere struct {
	BaseProvider `mapstructure:",squash"`
}

type Groq struct {
	BaseProvider `mapstructure:",squash"`
}

type Mistral struct {
	BaseProvider `mapstructure:",squash"`
}

type Ollama struct {
	BaseProvider `mapstructure:",squash"`
}

type OpenAI struct {
	BaseProvider `mapstructure:",squash"`
}

type Together struct {
	BaseProvider `mapstructure:",squash"`
}

// Command represents a named command with overrideable settings
type Command struct {
	Description     string `mapstructure:"description"`
	SystemPrompt    string `mapstructure:"system_prompt"`
	MessageTemplate string `mapstructure:"message_template"`

	ModelType string `toml:"model_type,omitempty"` // allows local-deep, etc

	// generation params
	Temperature *float64 `mapstructure:"temperature"`
	MaxTokens   *int     `mapstructure:"max_tokens"`

	// context (supports both direct and file-based)
	Context      string   `mapstructure:"context"`       // direct context string (supports multiline)
	ContextFiles []string `mapstructure:"context_files"` // file paths to include

	// exit code config
	ExitCodeMap string `mapstructure:"exit_code_map"` // exit code map name
}

// ReservedCommands are command names that cannot be overridden by users
var ReservedCommands = map[string]bool{
	"help":    true,
	"list":    true,
	"version": true,
	"config":  true,
	"set":     true,
}
