package prompt

import (
	"fmt"
	"strings"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type taskPromptProvider struct{}

func init() {
	source.Register("prompt", "prompt", taskPromptProvider{})
}

func (taskPromptProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	promptText := strings.TrimSpace(sourceRef.Reference)
	if promptText == "" {
		return workitem.WorkItem{}, fmt.Errorf("prompt text cannot be empty")
	}

	identifier := workitem.NormalizeIdentifier(promptText)
	if identifier == "" {
		return workitem.WorkItem{}, fmt.Errorf("prompt text does not contain a valid identifier")
	}

	return workitem.WorkItem{
		Mode:        workitem.ModeImplement,
		Source:      sourceRef,
		Identifier:  identifier,
		Title:       promptText,
		Description: promptText,
	}, nil
}
