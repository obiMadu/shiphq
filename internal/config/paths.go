package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigDir returns the XDG-compliant config directory for wtmag
// Uses XDG_CONFIG_HOME if set, otherwise ~/.config
func GetConfigDir() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		return filepath.Join(xdgConfig, "wtmag")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current dir if we can't get home
		return "wtmag"
	}

	return filepath.Join(home, ".config", "wtmag")
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.toml")
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	return os.MkdirAll(GetConfigDir(), 0755)
}

// EnsureConfigFile creates the config file with the provided contents if it does not exist yet.
func EnsureConfigFile(contents []byte) error {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect config file %s: %w", configPath, err)
	}

	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("failed to create config file %s: %w", configPath, err)
	}
	defer configFile.Close()

	if _, err := configFile.Write(contents); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}

// GetStateDir returns the XDG-compliant state directory for wtmag
// Uses XDG_STATE_HOME if set, otherwise ~/.local/state
func GetStateDir() string {
	xdgState := os.Getenv("XDG_STATE_HOME")
	if xdgState != "" {
		return filepath.Join(xdgState, "wtmag")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "wtmag-state"
	}

	return filepath.Join(home, ".local", "state", "wtmag")
}

// GetSessionStateDir returns the directory used for session metadata
func GetSessionStateDir() string {
	return filepath.Join(GetStateDir(), "sessions")
}

// EnsureSessionStateDir creates the session metadata directory if needed
func EnsureSessionStateDir() error {
	return os.MkdirAll(GetSessionStateDir(), 0755)
}
