package model

import (
	"fmt"
	"strings"
)

type opencodeTranslator struct{}

func (opencodeTranslator) commandArgs(selection Selection) ([]string, error) {
	commandArgs := []string{"--model", selection.ID()}
	translatedThinking, err := opencodeThinkingLevel(selection.Thinking)
	if err != nil {
		return nil, err
	}
	if translatedThinking != "" {
		commandArgs = append(commandArgs, "--variant", translatedThinking)
	}

	return commandArgs, nil
}

func (opencodeTranslator) supportsThinkingLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "off", "none", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func opencodeThinkingLevel(thinking string) (string, error) {
	switch normalizedThinking := strings.ToLower(strings.TrimSpace(thinking)); normalizedThinking {
	case "":
		return "", nil
	case "off":
		return "none", nil
	case "none", "minimal", "low", "medium", "high", "xhigh", "max":
		return normalizedThinking, nil
	default:
		return "", fmt.Errorf("unsupported thinking level %q for opencode (use off, none, minimal, low, medium, high, xhigh, or max)", thinking)
	}
}
