package agent

import (
	"fmt"
	"strings"

	agentmodel "github.com/obiMadu/wtmag/internal/agent/model"
	"github.com/obiMadu/wtmag/internal/config"
)

// Agent represents an AI coding agent that can be spawned in sessions
type Agent struct {
	Command    string   `toml:"command"`     // Base command (e.g., "opencode", "claude", "aider")
	Args       []string `toml:"args"`        // Static arguments before prompt delivery (e.g., ["--mode", "text"])
	PromptFlag string   `toml:"prompt_flag"` // How to pass prompt: "--prompt", "--message", "" (for positional)
}

// BuildCommand builds a shell-safe command string for tmux.
func (a Agent) BuildCommand(prompt string, modelSelection *agentmodel.Selection) (string, error) {
	parts := []string{shellQuote(a.Command)}
	for _, arg := range a.Args {
		parts = append(parts, shellQuote(arg))
	}

	modelArgs, err := agentmodel.CommandArgs(a.Command, modelSelection)
	if err != nil {
		return "", err
	}
	for _, arg := range modelArgs {
		parts = append(parts, shellQuote(arg))
	}

	if a.PromptFlag != "" {
		parts = append(parts, shellQuote(a.PromptFlag))
	}
	parts = append(parts, shellQuote(prompt))
	return strings.Join(parts, " "), nil
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

// DefaultName returns the configured default agent name.
func DefaultName() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	configuredDefault := strings.TrimSpace(cfg.DefaultAgent.Name)
	if configuredDefault == "" {
		return "", fmt.Errorf("agents.default.name is not configured in %s", config.LookupDescription())
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

	cfg, err := config.Load()
	if err != nil {
		return Agent{}, err
	}

	return lookup(trimmedName, cfg)
}

// List returns all configured agent names.
func List() []string {
	cfg, err := config.Load()
	if err != nil || cfg.Agents == nil {
		return nil
	}

	names := make([]string, 0, len(cfg.Agents))
	for name := range cfg.Agents {
		names = append(names, name)
	}

	return names
}

func lookup(name string, cfg *config.Config) (Agent, error) {
	if cfg != nil && cfg.Agents != nil {
		if configuredAgent, ok := cfg.Agents[name]; ok {
			agent := Agent{
				Command:    configuredAgent.Command,
				Args:       configuredAgent.Args,
				PromptFlag: configuredAgent.PromptFlag,
			}
			if err := agent.Validate(); err != nil {
				return Agent{}, fmt.Errorf("invalid agent config for '%s': %w", name, err)
			}
			return agent, nil
		}
	}

	return Agent{}, fmt.Errorf("unknown agent: %s (define it under [agents.%s] in %s)", name, name, config.LookupDescription())
}
