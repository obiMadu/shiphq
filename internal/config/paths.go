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

// GetStateDir returns the XDG-compliant state directory for shiphq
// Uses XDG_STATE_HOME if set, otherwise ~/.local/state
func GetStateDir() string {
	xdgState := os.Getenv("XDG_STATE_HOME")
	if xdgState != "" {
		return filepath.Join(xdgState, "shiphq")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "shiphq-state"
	}

	return filepath.Join(home, ".local", "state", "shiphq")
}

// GetSessionStateDir returns the directory used for session metadata
func GetSessionStateDir() string {
	return filepath.Join(GetStateDir(), "sessions")
}

// EnsureSessionStateDir creates the session metadata directory if needed
func EnsureSessionStateDir() error {
	return os.MkdirAll(GetSessionStateDir(), 0755)
}
