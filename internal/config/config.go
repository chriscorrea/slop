package config

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

//go:embed data/default_config.toml
var defaultConfigTOML string

//go:embed data/default_commands.toml
var defaultCommandsTOML string

// Manager handles configuration loading and management
type Manager struct {
	v      *viper.Viper
	cfg    *Config
	logger *slog.Logger
}

// NewManager creates a new configuration manager with default settings
func NewManager() *Manager {
	v := viper.New()

	// Register aliases for easier API key management
	v.RegisterAlias("mistral-key", "providers.mistral.api_key")
	v.RegisterAlias("cohere-key", "providers.cohere.api_key")
	v.RegisterAlias("anthropic-key", "providers.anthropic.api_key")

	// Bind provider API keys to intuitive environment variable names
	_ = v.BindEnv("providers.mistral.api_key", "MISTRAL_API_KEY")
	_ = v.BindEnv("providers.cohere.api_key", "COHERE_API_KEY")
	_ = v.BindEnv("providers.anthropic.api_key", "ANTHROPIC_API_KEY")
	_ = v.BindEnv("providers.openai.api_key", "OPENAI_API_KEY")
	_ = v.BindEnv("providers.groq.api_key", "GROQ_API_KEY")

	return &Manager{
		v:   v,
		cfg: &Config{}, // empty config, defaults loaded from embedded TOML in Load()
	}
}

// WithLogger sets the logger for the configuration manager
func (m *Manager) WithLogger(logger *slog.Logger) *Manager {
	m.logger = logger
	return m
}

// Load loads configuration from the specified TOML file, merging with defaults
func (m *Manager) Load(configPath string) error {
	if m.logger != nil {
		m.logger.Debug("Attempting to load config file", "path", configPath)
	}

	// Set config type for TOML
	m.v.SetConfigType("toml")

	// Load defaults from embedded TOML
	err := m.v.ReadConfig(strings.NewReader(defaultConfigTOML))
	if err != nil {
		return fmt.Errorf("failed to load embedded defaults: %w", err)
	}

	// Set config file
	m.v.SetConfigFile(configPath)

	// Merge user config file over defaults
	err = m.v.MergeInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		var pathError *os.PathError
		if !errors.As(err, &configFileNotFoundError) && !errors.As(err, &pathError) {
			return err
		}
		// check "no such file or directory"
		if pathError != nil && !os.IsNotExist(pathError) {
			return err
		}
		// file doesn't exist, but we still want to be able to write to it
		// so we set the config file path again to ensure Viper knows where to write
		if m.logger != nil {
			m.logger.Debug("Config file not found")
		}

		// auto-create default config file if it doesn't exist
		if err := m.createDefaultConfigFile(configPath); err != nil {
			return fmt.Errorf("failed to create default config file: %w", err)
		}

		m.v.SetConfigFile(configPath)

	} else if m.logger != nil {
		m.logger.Info("Configuration loaded successfully", "path", m.v.ConfigFileUsed())
	}

	// Unmarshal final configuration
	err = m.v.Unmarshal(&m.cfg)
	if err != nil {
		return err
	}

	// init commands map and load defaults from embedded TOML
	if m.cfg.Commands == nil {
		m.cfg.Commands = make(map[string]Command)
	}

	// load default commands from embedded TOML
	commandsV := viper.New()
	commandsV.SetConfigType("toml")
	if err := commandsV.ReadConfig(strings.NewReader(defaultCommandsTOML)); err != nil {
		return fmt.Errorf("failed to load embedded default commands: %w", err)
	}

	var defaultCommands map[string]Command
	if err := commandsV.UnmarshalKey("commands", &defaultCommands); err != nil {
		return fmt.Errorf("failed to unmarshal embedded default commands: %w", err)
	}

	// add default commands to config
	for name, cmd := range defaultCommands {
		m.cfg.Commands[name] = cmd
	}

	// load user commands from commands.toml
	configDir := filepath.Dir(configPath)
	commandsPath := filepath.Join(configDir, "commands.toml")

	// auto-create default commands file if it doesn't exist
	if err := m.createDefaultCommandsFile(commandsPath); err != nil {
		return fmt.Errorf("failed to create default commands file: %w", err)
	}

	if err := m.loadUserCommands(commandsPath); err != nil {
		if m.logger != nil {
			m.logger.Warn("Failed to load user commands", "error", err)
		}
	}

	// validate no reserved keywords are overridden
	if err := m.validateCommands(); err != nil {
		return fmt.Errorf("invalid command configuration: %w", err)
	}

	// post-process configuration to handle special cases
	m.postProcessConfig()

	return nil
}

// config returns the current configuration
func (m *Manager) Config() *Config {
	return m.cfg
}

// Viper returns the underlying Viper instance for flag binding
func (m *Manager) Viper() *viper.Viper {
	return m.v
}

// save writes the current configuration state back to the config file
func (m *Manager) Save() error {
	// get the config file path
	configFile := m.v.ConfigFileUsed()

	if configFile == "" {
		return fmt.Errorf("no config file path set")
	}

	// ensure the directory exists
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// check if file exists to decide between SafeWriteConfigAs and WriteConfigAs
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// file doesn't exist, use SafeWriteConfigAs
		if err := m.v.SafeWriteConfigAs(configFile); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	} else {
		// file exists, overwrite with WriteConfigAs
		if err := m.v.WriteConfigAs(configFile); err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
	}

	// reload the configuration struct to reflect the changes
	if err := m.v.Unmarshal(&m.cfg); err != nil {
		return fmt.Errorf("failed to reload configuration after save: %w", err)
	}

	return nil
}

// NewDefaultFromEmbedded creates a Config struct populated from embedded TOML
// note we're primarily using this for testing
func NewDefaultFromEmbedded() *Config {
	v := viper.New()
	v.SetConfigType("toml")

	// load defaults from embedded TOML
	if err := v.ReadConfig(strings.NewReader(defaultConfigTOML)); err != nil {
		panic(fmt.Sprintf("failed to load embedded defaults in test helper: %v", err))
	}

	// load commands from embedded TOML using a separate viper instance
	commandsV := viper.New()
	commandsV.SetConfigType("toml")
	if err := commandsV.ReadConfig(strings.NewReader(defaultCommandsTOML)); err != nil {
		panic(fmt.Sprintf("failed to load embedded commands in test helper: %v", err))
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		panic(fmt.Sprintf("failed to unmarshal embedded config in test helper: %v", err))
	}

	// load commands separately
	var commands map[string]Command
	if err := commandsV.UnmarshalKey("commands", &commands); err != nil {
		panic(fmt.Sprintf("failed to unmarshal embedded commands in test helper: %v", err))
	}
	cfg.Commands = commands

	return cfg
}

// loadUserCommands loads commands from commands.toml and merges with defaults
func (m *Manager) loadUserCommands(commandsPath string) error {
	commandsViper := viper.New()
	commandsViper.SetConfigFile(commandsPath)

	if err := commandsViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // commands file is optional
		}
		return fmt.Errorf("failed to read commands config: %w", err)
	}

	userCommands := commandsViper.GetStringMap("commands")
	if len(userCommands) == 0 {
		return nil // no commands defined
	}

	// convert the map to our Command struct
	var commands map[string]Command
	if err := commandsViper.UnmarshalKey("commands", &commands); err != nil {
		return fmt.Errorf("failed to unmarshal commands: %w", err)
	}

	// merge user commands with defaults (user commands can override defaults)
	for name, cmd := range commands {
		m.cfg.Commands[name] = cmd
	}

	return nil
}

// validateCommands ensures no reserved keywords are overridden
func (m *Manager) validateCommands() error {
	for cmdName := range m.cfg.Commands {
		if ReservedCommands[cmdName] {
			return fmt.Errorf("cannot override reserved command: %s", cmdName)
		}
	}
	return nil
}

// postProcessConfig handles special processing after configuration loading
func (m *Manager) postProcessConfig() {
	// handle seed parameter: convert 0 to nil (no seed)
	if m.v.IsSet("parameters.seed") {
		seedValue := m.v.GetInt("parameters.seed")
		if seedValue == 0 {
			m.cfg.Parameters.Seed = nil
		} else {
			m.cfg.Parameters.Seed = &seedValue
		}
	}
}

// WithCommandOverrides creates a new Config with command overrides applied
func (c *Config) WithCommandOverrides(cmd Command) *Config {
	// create a deep copy of the config to avoid mutation
	newConfig := *c
	newConfig.Parameters = c.Parameters // cpy struct

	// apply command overrides with precedence
	if cmd.SystemPrompt != "" {
		newConfig.Parameters.SystemPrompt = cmd.SystemPrompt
	}
	if cmd.Temperature != nil {
		newConfig.Parameters.Temperature = *cmd.Temperature
	}
	if cmd.MaxTokens != nil {
		newConfig.Parameters.MaxTokens = *cmd.MaxTokens
	}

	return &newConfig
}

// createDefaultConfigFile creates the default config.toml file if it doesn't exist
func (m *Manager) createDefaultConfigFile(configPath string) error {
	// check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // nothing to do here!
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check config file: %w", err)
	}

	// create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// write default config content
	if err := os.WriteFile(configPath, []byte(defaultConfigTOML), 0600); err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}

	// let user know where the config file is
	fmt.Fprintf(os.Stderr, "Created default config.toml at %s\n", configPath)
	// encourage guided setup
	fmt.Fprintf(os.Stderr, "For a guided setup, run: slop init\n")

	if m.logger != nil {
		m.logger.Info("Created default config file", "path", configPath)
	}

	return nil
}

// createDefaultCommandsFile creates the default commands.toml file if it doesn't exist
func (m *Manager) createDefaultCommandsFile(commandsPath string) error {
	// check if command file already exists
	if _, err := os.Stat(commandsPath); err == nil {
		return nil // nothing to do here
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check commands file: %w", err)
	}

	// create directory if it doesn't exist
	commandsDir := filepath.Dir(commandsPath)
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// write default commands content
	if err := os.WriteFile(commandsPath, []byte(defaultCommandsTOML), 0600); err != nil {
		return fmt.Errorf("failed to write default commands file: %w", err)
	}

	if m.logger != nil {
		m.logger.Info("Created default commands file", "path", commandsPath)
	}

	return nil
}
