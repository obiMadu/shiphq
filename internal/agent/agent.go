package agent

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/obiMadu/shiphq/internal/config"
)

// Agent represents an AI coding agent that can be spawned in sessions
type Agent struct {
	Command    string `toml:"command"`     // Base command (e.g., "opencode", "claude", "aider")
	PromptFlag string `toml:"prompt_flag"` // How to pass prompt: "--prompt", "--message", "" (for positional)
}

// BuildCommand builds the full command string with the prompt
func (a Agent) BuildCommand(prompt string) string {
	if a.PromptFlag == "" {
		// Positional argument (e.g., claude "...", codex "...")
		return fmt.Sprintf("%s %q", a.Command, prompt)
	}
	// Flag-based (e.g., opencode --prompt "...", aider --message "...")
	return fmt.Sprintf("%s %s %q", a.Command, a.PromptFlag, prompt)
}

// Validate checks if the agent configuration is valid
func (a Agent) Validate() error {
	if a.Command == "" {
		return fmt.Errorf("agent command cannot be empty")
	}
	// PromptFlag can be empty (positional args are valid)
	return nil
}

// Built-in agents registry - easy for maintainers to add new ones here
var builtinAgents = map[string]Agent{
	"opencode": {
		Command:    "opencode",
		PromptFlag: "--prompt",
	},
	"claude": {
		Command:    "claude",
		PromptFlag: "", // Positional: claude "..."
	},
	"codex": {
		Command:    "codex",
		PromptFlag: "", // Positional: codex "..."
	},
}

// Config represents user-defined agent configuration
type Config struct {
	Agents map[string]Agent `toml:"agents"`
}

// Get retrieves an agent by name
// Checks built-in first, then user config
func Get(name string) (Agent, error) {
	// First check built-in agents
	if agent, ok := builtinAgents[name]; ok {
		return agent, nil
	}

	// Then check user config
	cfg, err := loadUserConfig()
	if err == nil && cfg.Agents != nil {
		if agent, ok := cfg.Agents[name]; ok {
			if err := agent.Validate(); err != nil {
				return Agent{}, fmt.Errorf("invalid agent config for '%s': %w", name, err)
			}
			return agent, nil
		}
	}

	return Agent{}, fmt.Errorf("unknown agent: %s (supported built-in: opencode, claude, codex)", name)
}

// List returns all supported agent names (built-in + user-defined)
func List() []string {
	names := make([]string, 0, len(builtinAgents))
	for name := range builtinAgents {
		names = append(names, name)
	}

	// Add user-defined agents
	cfg, err := loadUserConfig()
	if err == nil && cfg.Agents != nil {
		for name := range cfg.Agents {
			// Skip if already in built-in
			if _, ok := builtinAgents[name]; !ok {
				names = append(names, name)
			}
		}
	}

	return names
}

// loadUserConfig loads user-defined agent configurations
func loadUserConfig() (*Config, error) {
	configPath := config.GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode agent config: %w", err)
	}

	return &cfg, nil
}
