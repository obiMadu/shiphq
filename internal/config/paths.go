package config

import (
	"os"
	"path/filepath"
)

// GetConfigDir returns the XDG-compliant config directory for shiphq
// Uses XDG_CONFIG_HOME if set, otherwise ~/.config
func GetConfigDir() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		return filepath.Join(xdgConfig, "shiphq")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current dir if we can't get home
		return "shiphq"
	}

	return filepath.Join(home, ".config", "shiphq")
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.toml")
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	return os.MkdirAll(GetConfigDir(), 0755)
}
