package model

import (
	"fmt"
	"strings"
)

type piTranslator struct{}

func (piTranslator) commandArgs(selection Selection) ([]string, error) {
	translatedThinking, err := piThinkingLevel(selection.Thinking)
	if err != nil {
		return nil, err
	}

	modelValue := selection.ID()
	if translatedThinking != "" {
		modelValue += ":" + translatedThinking
	}

	return []string{"--model", modelValue}, nil
}

func (piTranslator) supportsThinkingLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "off", "none", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func piThinkingLevel(thinking string) (string, error) {
	switch normalizedThinking := strings.ToLower(strings.TrimSpace(thinking)); normalizedThinking {
	case "":
		return "", nil
	case "none":
		return "off", nil
	case "max":
		return "xhigh", nil
	case "off", "minimal", "low", "medium", "high", "xhigh":
		return normalizedThinking, nil
	default:
		return "", fmt.Errorf("unsupported thinking level %q for pi (use off, minimal, low, medium, high, xhigh, or max)", thinking)
	}
}
