package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	defaultImplementationLaunch = "window"
	defaultReviewLaunch         = "window"
)

type Config struct {
	DefaultAgent DefaultAgentConfig
	Agents       map[string]AgentConfig
	Launch       LaunchConfig
}

type DefaultAgentConfig struct {
	Name string `toml:"name"`
}

type AgentConfig struct {
	Command    string   `toml:"command"`
	Args       []string `toml:"args"`
	PromptFlag string   `toml:"prompt_flag"`
}

type LaunchConfig struct {
	Implementation string `toml:"implementation"`
	Review         string `toml:"review"`
}

func Load() (*Config, error) {
	mergedConfig := defaultConfig()

	globalConfig, err := loadConfigFile(GetGlobalConfigPath())
	if err != nil {
		return nil, err
	}
	mergedConfig = mergeConfig(mergedConfig, globalConfig)

	projectConfigPath, err := FindProjectConfigPath()
	if err != nil {
		return nil, err
	}

	projectConfig, err := loadConfigFile(projectConfigPath)
	if err != nil {
		return nil, err
	}

	return mergeConfig(mergedConfig, projectConfig), nil
}

func LookupDescription() string {
	configPaths, err := GetConfigLookupPaths()
	if err != nil || len(configPaths) == 0 {
		return GetGlobalConfigPath()
	}

	return strings.Join(configPaths, " or ")
}

func defaultConfig() *Config {
	return &Config{
		Launch: LaunchConfig{
			Implementation: defaultImplementationLaunch,
			Review:         defaultReviewLaunch,
		},
	}
}

func loadConfigFile(configPath string) (*Config, error) {
	if strings.TrimSpace(configPath) == "" {
		return &Config{}, nil
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to inspect config file %s: %w", configPath, err)
	}

	type rawConfig struct {
		Agents map[string]toml.Primitive `toml:"agents"`
		Launch LaunchConfig              `toml:"launch"`
	}

	var rawCfg rawConfig
	metadata, err := toml.DecodeFile(configPath, &rawCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config %s: %w", configPath, err)
	}

	cfg := &Config{
		Launch: rawCfg.Launch,
	}

	for name, primitive := range rawCfg.Agents {
		if name == "default" {
			if err := metadata.PrimitiveDecode(primitive, &cfg.DefaultAgent); err != nil {
				return nil, fmt.Errorf("failed to decode agents.default in %s: %w", configPath, err)
			}
			continue
		}

		var configuredAgent AgentConfig
		if err := metadata.PrimitiveDecode(primitive, &configuredAgent); err != nil {
			return nil, fmt.Errorf("failed to decode agents.%s in %s: %w", name, configPath, err)
		}

		if cfg.Agents == nil {
			cfg.Agents = make(map[string]AgentConfig)
		}
		cfg.Agents[name] = configuredAgent
	}

	return cfg, nil
}

func mergeConfig(baseConfig, overrideConfig *Config) *Config {
	mergedConfig := &Config{}

	if baseConfig != nil {
		mergedConfig.DefaultAgent = baseConfig.DefaultAgent
		mergedConfig.Launch = baseConfig.Launch
		if len(baseConfig.Agents) != 0 {
			mergedConfig.Agents = make(map[string]AgentConfig, len(baseConfig.Agents))
			for name, configuredAgent := range baseConfig.Agents {
				mergedConfig.Agents[name] = configuredAgent
			}
		}
	}

	if overrideConfig == nil {
		return mergedConfig
	}

	if strings.TrimSpace(overrideConfig.DefaultAgent.Name) != "" {
		mergedConfig.DefaultAgent = overrideConfig.DefaultAgent
	}

	if trimmedImplementation := strings.TrimSpace(overrideConfig.Launch.Implementation); trimmedImplementation != "" {
		mergedConfig.Launch.Implementation = trimmedImplementation
	}

	if trimmedReview := strings.TrimSpace(overrideConfig.Launch.Review); trimmedReview != "" {
		mergedConfig.Launch.Review = trimmedReview
	}

	if len(overrideConfig.Agents) != 0 {
		if mergedConfig.Agents == nil {
			mergedConfig.Agents = make(map[string]AgentConfig, len(overrideConfig.Agents))
		}

		for name, configuredAgent := range overrideConfig.Agents {
			mergedConfig.Agents[name] = configuredAgent
		}
	}

	return mergedConfig
}
