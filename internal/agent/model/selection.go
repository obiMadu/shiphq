package model

import (
	"fmt"
	"strings"
)

// Selection is the wtmag-level model override format.
type Selection struct {
	Provider string
	Model    string
	Thinking string
}

// ParseSelection parses provider/model[:thinking] values.
func ParseSelection(value string) (*Selection, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil, nil
	}

	providerSeparator := strings.Index(trimmedValue, "/")
	if providerSeparator <= 0 || providerSeparator == len(trimmedValue)-1 {
		return nil, fmt.Errorf("model must use provider/model[:thinking] format")
	}

	selection := &Selection{
		Provider: strings.TrimSpace(trimmedValue[:providerSeparator]),
	}
	modelAndThinking := strings.TrimSpace(trimmedValue[providerSeparator+1:])
	if strings.HasSuffix(modelAndThinking, ":") {
		return nil, fmt.Errorf("thinking level cannot be empty in model override %q", trimmedValue)
	}

	thinkingSeparator := strings.LastIndex(modelAndThinking, ":")
	if thinkingSeparator >= 0 && isKnownThinkingLevel(modelAndThinking[thinkingSeparator+1:]) {
		selection.Model = strings.TrimSpace(modelAndThinking[:thinkingSeparator])
		selection.Thinking = strings.TrimSpace(modelAndThinking[thinkingSeparator+1:])
	} else {
		selection.Model = modelAndThinking
	}

	if selection.Provider == "" || selection.Model == "" {
		return nil, fmt.Errorf("model must use provider/model[:thinking] format")
	}

	return selection, nil
}

func (selection Selection) ID() string {
	return selection.Provider + "/" + selection.Model
}
