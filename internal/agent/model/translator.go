package model

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type translator interface {
	commandArgs(selection Selection) ([]string, error)
	supportsThinkingLevel(level string) bool
}

var translators = map[string]translator{
	"opencode": opencodeTranslator{},
	"pi":       piTranslator{},
}

func CommandArgs(command string, selection *Selection) ([]string, error) {
	if selection == nil {
		return nil, nil
	}

	resolvedTranslator, ok := translatorForCommand(command)
	if !ok {
		return nil, fmt.Errorf("wtmag model overrides are currently supported only for %s agents; command %q is not supported", supportedAgents(), command)
	}

	return resolvedTranslator.commandArgs(*selection)
}

func translatorForCommand(command string) (translator, bool) {
	commandName := filepath.Base(strings.TrimSpace(command))
	resolvedTranslator, ok := translators[commandName]
	return resolvedTranslator, ok
}

func supportedAgents() string {
	agentNames := make([]string, 0, len(translators))
	for name := range translators {
		agentNames = append(agentNames, name)
	}
	sort.Strings(agentNames)

	return strings.Join(agentNames, " and ")
}

func isKnownThinkingLevel(value string) bool {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	if normalizedValue == "" {
		return false
	}

	for _, resolvedTranslator := range translators {
		if resolvedTranslator.supportsThinkingLevel(normalizedValue) {
			return true
		}
	}

	return false
}
