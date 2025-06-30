package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ConfigFieldInfo contains metadata about a configuration field
type ConfigFieldInfo struct {
	Type        reflect.Type
	Description string
	Default     interface{}
	Validation  func(interface{}) error
}

// ConfigSchema holds the registry of valid configuration paths and aliases
type ConfigSchema struct {
	ValidPaths map[string]ConfigFieldInfo
	Aliases    map[string]string
}

// validateFloat64Range returns a validation function for float64 values within a range
func validateFloat64Range(min, max float64) func(interface{}) error {
	return func(value interface{}) error {
		if v, ok := value.(float64); ok {
			if v < min || v > max {
				return fmt.Errorf("value must be between %.2f and %.2f", min, max)
			}
			return nil
		}
		return fmt.Errorf("expected float64, got %T", value)
	}
}

// validateIntRange returns a validation function for int values within a range
func validateIntRange(min, max int) func(interface{}) error {
	return func(value interface{}) error {
		if v, ok := value.(int); ok {
			if v < min || v > max {
				return fmt.Errorf("value must be between %d and %d", min, max)
			}
			return nil
		}
		return fmt.Errorf("expected int, got %T", value)
	}
}

// validateOptionalInt returns a validation function for optional int values (can be nil)
func validateOptionalInt() func(interface{}) error {
	return func(value interface{}) error {
		if value == nil {
			return nil // nil is allowed for optional parameters
		}
		if _, ok := value.(int); ok {
			return nil
		}
		return fmt.Errorf("expected int or nil, got %T", value)
	}
}

// DefaultConfigSchema returns the default configuration schema
func DefaultConfigSchema() *ConfigSchema {
	return &ConfigSchema{
		ValidPaths: map[string]ConfigFieldInfo{
			// Parameters
			"parameters.temperature": {
				Type:        reflect.TypeOf(float64(0)),
				Description: "LLM temperature for response randomness (0.0-2.0)",
				Default:     0.7,
				Validation:  validateFloat64Range(0.0, 1.0),
			},
			"parameters.max_tokens": {
				Type:        reflect.TypeOf(int(0)),
				Description: "Maximum number of tokens in LLM response",
				Default:     2048,
				Validation:  validateIntRange(1, 100000),
			},
			"parameters.max_retries": {
				Type:        reflect.TypeOf(int(0)),
				Description: "Maximum number of retry attempts for failed requests (max: 5)",
				Default:     2,
				Validation:  validateIntRange(0, 5),
			},
			"parameters.top_p": {
				Type:        reflect.TypeOf(float64(0)),
				Description: "Top P sampling for LLM responses (0.0-1.0)",
				Default:     1.0,
				Validation:  validateFloat64Range(0.0, 1.0),
			},
			"parameters.system_prompt": {
				Type:        reflect.TypeOf(""),
				Description: "Default system prompt for LLM inference",
				Default:     "",
			},
			"parameters.timeout": {
				Type:        reflect.TypeOf(int(0)),
				Description: "Timeout in seconds for LLM requests",
				Default:     60,
				Validation:  validateIntRange(1, 600),
			},
			"parameters.stream": {
				Type:        reflect.TypeOf(bool(false)),
				Description: "Enable streaming responses from LLM",
				Default:     false,
			},
			"parameters.seed": {
				Type:        reflect.TypeOf((*int)(nil)).Elem(),
				Description: "Random seed for deterministic LLM outputs (optional)",
				Default:     nil,
				Validation:  validateOptionalInt(),
			},
			"parameters.default_model_type": {
				Type:        reflect.TypeOf(""),
				Description: "Default model type preference (fast/deep)",
				Default:     "fast",
			},
			"parameters.default_location": {
				Type:        reflect.TypeOf(""),
				Description: "Default model location preference (local/remote)",
				Default:     "remote",
			},

			// Provider API Keys
			"providers.anthropic.api_key": {
				Type:        reflect.TypeOf(""),
				Description: "Anthropic API key for Claude models",
				Default:     "",
			},
			"providers.openai.api_key": {
				Type:        reflect.TypeOf(""),
				Description: "OpenAI API key for GPT models",
				Default:     "",
			},
			"providers.cohere.api_key": {
				Type:        reflect.TypeOf(""),
				Description: "Cohere API key for Command models",
				Default:     "",
			},
			"providers.anthropic.base_url": {
				Type:        reflect.TypeOf(""),
				Description: "Anthropic API base URL",
				Default:     "https://api.anthropic.com/v1",
			},
			"providers.openai.base_url": {
				Type:        reflect.TypeOf(""),
				Description: "OpenAI API base URL",
				Default:     "https://api.openai.com/v1",
			},
			"providers.cohere.base_url": {
				Type:        reflect.TypeOf(""),
				Description: "Cohere API base URL",
				Default:     "https://api.cohere.com/v2",
			},
			"providers.ollama.base_url": {
				Type:        reflect.TypeOf(""),
				Description: "Ollama API base URL",
				Default:     "http://127.0.0.1:11434",
			},
			"providers.mistral.api_key": {
				Type:        reflect.TypeOf(""),
				Description: "Mistral API key for Mistral models",
				Default:     "",
			},
			"providers.mistral.base_url": {
				Type:        reflect.TypeOf(""),
				Description: "Mistral API base URL",
				Default:     "https://api.mistral.ai/v1",
			},

			// Model configurations
			"models.remote.fast.provider": {
				Type:        reflect.TypeOf(""),
				Description: "Provider for remote fast model",
				Default:     "anthropic",
			},
			"models.remote.fast.name": {
				Type:        reflect.TypeOf(""),
				Description: "Name of remote fast model",
				Default:     "claude-3-5-haiku-latest",
			},
			"models.remote.deep.provider": {
				Type:        reflect.TypeOf(""),
				Description: "Provider for remote deep/reasoning model",
				Default:     "anthropic",
			},
			"models.remote.deep.name": {
				Type:        reflect.TypeOf(""),
				Description: "Name of remote deep/reasoning model",
				Default:     "claude-sonnet-4-20250514",
			},
			"models.local.fast.provider": {
				Type:        reflect.TypeOf(""),
				Description: "Provider for local fast model",
				Default:     "ollama",
			},
			"models.local.fast.name": {
				Type:        reflect.TypeOf(""),
				Description: "Name of local fast model",
				Default:     "gemma3:latest",
			},
			"models.local.deep.provider": {
				Type:        reflect.TypeOf(""),
				Description: "Provider for local deep/reasoning model",
				Default:     "ollama",
			},
			"models.local.deep.name": {
				Type:        reflect.TypeOf(""),
				Description: "Name of local deep/reasoning model",
				Default:     "deepseek-r1:14b",
			},

			// Format options
			"format.json": {
				Type:        reflect.TypeOf(bool(false)),
				Description: "Format response as JSON",
				Default:     false,
			},
			"format.yaml": {
				Type:        reflect.TypeOf(bool(false)),
				Description: "Format response as YAML",
				Default:     false,
			},
			"format.md": {
				Type:        reflect.TypeOf(bool(false)),
				Description: "Format response as Markdown",
				Default:     false,
			},
			"format.xml": {
				Type:        reflect.TypeOf(bool(false)),
				Description: "Format response as XML",
				Default:     false,
			},
		},

		Aliases: map[string]string{
			// parameter aliases
			"temperature":   "parameters.temperature",
			"temp":          "parameters.temperature",
			"max-tokens":    "parameters.max_tokens",
			"max-retries":   "parameters.max_retries",
			"top-p":         "parameters.top_p",
			"system":        "parameters.system_prompt",
			"system-prompt": "parameters.system_prompt",
			"timeout":       "parameters.timeout",
			// "stream":             "parameters.stream",
			"default-model-type": "parameters.default_model_type",
			"default-location":   "parameters.default_location",
			"seed":               "parameters.seed",

			// provider api key aliases
			"anthropic-key": "providers.anthropic.api_key",
			"openai-key":    "providers.openai.api_key",
			"cohere-key":    "providers.cohere.api_key",
			"mistral-key":   "providers.mistral.api_key",

			// local provider endpoint
			"ollama-url": "providers.ollama.base_url",

			// quick set providers/models
			"remote-fast-provider": "models.remote.fast.provider",
			"remote-fast-model":    "models.remote.fast.name",
			"remote-deep-provider": "models.remote.deep.provider",
			"remote-deep-model":    "models.remote.deep.name",
			"local-fast-provider":  "models.local.fast.provider",
			"local-fast-model":     "models.local.fast.name",
			"local-deep-provider":  "models.local.deep.provider",
			"local-deep-model":     "models.local.deep.name",

			// format aliases
			"json":     "format.json",
			"yaml":     "format.yaml",
			"markdown": "format.md",
			"md":       "format.md",
			"xml":      "format.xml",
		},
	}
}

// ResolveKey resolves an alias to its canonical path or returns the path if already canonical
func (s *ConfigSchema) ResolveKey(key string) (string, error) {
	// Check if it's an alias first
	if canonicalPath, exists := s.Aliases[key]; exists {
		return canonicalPath, nil
	}

	// Check if it's a valid direct path
	if _, exists := s.ValidPaths[key]; exists {
		return key, nil
	}

	// Return error with suggestions
	suggestions := s.FindSimilarKeys(key)
	if len(suggestions) > 0 {
		return "", fmt.Errorf("invalid config key %q. Did you mean one of: %s", key, strings.Join(suggestions, ", "))
	}

	return "", fmt.Errorf("invalid config key %q. Use 'slop config list' to see valid keys", key)
}

// ValidateValue validates a value against the field's type and validation rules
func (s *ConfigSchema) ValidateValue(path string, value interface{}) error {
	fieldInfo, exists := s.ValidPaths[path]
	if !exists {
		return fmt.Errorf("unknown config path: %s", path)
	}

	// Check type compatibility
	valueType := reflect.TypeOf(value)
	if valueType != fieldInfo.Type {
		return fmt.Errorf("expected %s, got %s", fieldInfo.Type.String(), valueType.String())
	}

	// Run custom validation if present
	if fieldInfo.Validation != nil {
		return fieldInfo.Validation(value)
	}

	return nil
}

// GetFieldInfo returns information about a configuration field
func (s *ConfigSchema) GetFieldInfo(path string) (ConfigFieldInfo, error) {
	fieldInfo, exists := s.ValidPaths[path]
	if !exists {
		return ConfigFieldInfo{}, fmt.Errorf("unknown config path: %s", path)
	}
	return fieldInfo, nil
}

// ListAllKeys returns all valid configuration keys (canonical paths and aliases)
func (s *ConfigSchema) ListAllKeys() []string {
	var keys []string

	// Add canonical paths
	for path := range s.ValidPaths {
		keys = append(keys, path)
	}

	// Add aliases
	for alias := range s.Aliases {
		keys = append(keys, alias)
	}

	sort.Strings(keys)
	return keys
}

// ListCanonicalKeys returns only the canonical configuration paths
func (s *ConfigSchema) ListCanonicalKeys() []string {
	var keys []string
	for path := range s.ValidPaths {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	return keys
}

// ListAliases returns only the alias keys
func (s *ConfigSchema) ListAliases() []string {
	var aliases []string
	for alias := range s.Aliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

// FindSimilarKeys finds keys similar to the input using simple string matching
func (s *ConfigSchema) FindSimilarKeys(key string) []string {
	var suggestions []string
	lowerKey := strings.ToLower(key)

	// Check canonical paths
	for path := range s.ValidPaths {
		if strings.Contains(strings.ToLower(path), lowerKey) ||
			strings.Contains(lowerKey, strings.ToLower(strings.Split(path, ".")[len(strings.Split(path, "."))-1])) {
			suggestions = append(suggestions, path)
		}
	}

	// Check aliases
	for alias := range s.Aliases {
		if strings.Contains(strings.ToLower(alias), lowerKey) ||
			strings.Contains(lowerKey, strings.ToLower(alias)) {
			suggestions = append(suggestions, alias)
		}
	}

	// Limit suggestions to avoid overwhelming output
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}
