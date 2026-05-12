package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/obiMadu/wtmag/internal/config"
)

// Agent represents an AI coding agent that can be spawned in sessions
type Agent struct {
	Command    string   `toml:"command"`     // Base command (e.g., "opencode", "claude", "aider")
	Args       []string `toml:"args"`        // Static arguments before prompt delivery (e.g., ["--mode", "text"])
	PromptFlag string   `toml:"prompt_flag"` // How to pass prompt: "--prompt", "--message", "" (for positional)
}

// BuildCommand builds a shell-safe command string for tmux.
func (a Agent) BuildCommand(prompt string) string {
	parts := []string{shellQuote(a.Command)}
	for _, arg := range a.Args {
		parts = append(parts, shellQuote(arg))
	}
	if a.PromptFlag != "" {
		parts = append(parts, shellQuote(a.PromptFlag))
	}
	parts = append(parts, shellQuote(prompt))
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// Validate checks if the agent configuration is valid
func (a Agent) Validate() error {
	if a.Command == "" {
		return fmt.Errorf("agent command cannot be empty")
	}
	// PromptFlag can be empty (positional args are valid)
	return nil
}

// Config represents agent-related user configuration.
type Config struct {
	DefaultAgent DefaultAgentConfig
	Agents       map[string]Agent
}

type DefaultAgentConfig struct {
	Name string `toml:"name"`
}

// DefaultName returns the configured default agent name.
func DefaultName() (string, error) {
	cfg, err := loadUserConfig()
	if err != nil {
		return "", err
	}

	configuredDefault := strings.TrimSpace(cfg.DefaultAgent.Name)
	if configuredDefault == "" {
		return "", fmt.Errorf("agents.default.name is not configured in %s", config.GetConfigPath())
	}

	if _, err := lookup(configuredDefault, cfg); err != nil {
		return "", fmt.Errorf("invalid agents.default.name %q: %w", configuredDefault, err)
	}

	return configuredDefault, nil
}

// Get retrieves an agent by name from config.
func Get(name string) (Agent, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Agent{}, fmt.Errorf("agent name cannot be empty")
	}

	cfg, err := loadUserConfig()
	if err != nil {
		return Agent{}, err
	}

	return lookup(trimmedName, cfg)
}

// List returns all configured agent names.
func List() []string {
	cfg, err := loadUserConfig()
	if err != nil || cfg.Agents == nil {
		return nil
	}

	names := make([]string, 0, len(cfg.Agents))
	for name := range cfg.Agents {
		names = append(names, name)
	}

	return names
}

// loadUserConfig loads agent-related user configuration
func loadUserConfig() (*Config, error) {
	configPath := config.GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	type rawConfig struct {
		Agents map[string]toml.Primitive `toml:"agents"`
	}

	var rawCfg rawConfig
	metadata, err := toml.DecodeFile(configPath, &rawCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode agent config: %w", err)
	}

	cfg := &Config{}
	for name, primitive := range rawCfg.Agents {
		if name == "default" {
			if err := metadata.PrimitiveDecode(primitive, &cfg.DefaultAgent); err != nil {
				return nil, fmt.Errorf("failed to decode agents.default: %w", err)
			}
			continue
		}

		var configuredAgent Agent
		if err := metadata.PrimitiveDecode(primitive, &configuredAgent); err != nil {
			return nil, fmt.Errorf("failed to decode agents.%s: %w", name, err)
		}

		if cfg.Agents == nil {
			cfg.Agents = make(map[string]Agent)
		}
		cfg.Agents[name] = configuredAgent
	}

	return cfg, nil
}

func lookup(name string, cfg *Config) (Agent, error) {
	if cfg != nil && cfg.Agents != nil {
		if agent, ok := cfg.Agents[name]; ok {
			if err := agent.Validate(); err != nil {
				return Agent{}, fmt.Errorf("invalid agent config for '%s': %w", name, err)
			}
			return agent, nil
		}
	}

	return Agent{}, fmt.Errorf("unknown agent: %s (define it under [agents.%s] in %s)", name, name, config.GetConfigPath())
}
