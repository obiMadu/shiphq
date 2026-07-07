package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/obiMadu/wtmag/internal/workitem"
)

type viewPayload struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
}

func runView(sourceRef workitem.SourceRef, args []string) (viewPayload, error) {
	output, err := exec.Command("gh", args...).CombinedOutput()
	if err != nil {
		return viewPayload{}, fmt.Errorf("failed to fetch GitHub %s %s: %w\n%s", sourceRef.Kind, sourceRef.Reference, err, output)
	}

	var payload viewPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return viewPayload{}, fmt.Errorf("failed to decode GitHub %s %s: %w", sourceRef.Kind, sourceRef.Reference, err)
	}

	return payload, nil
}

func baseWorkItem(sourceRef workitem.SourceRef, payload viewPayload) (workitem.WorkItem, error) {
	identifier := workitem.NormalizeIdentifier(sourceRef.Reference)
	if identifier == "" {
		return workitem.WorkItem{}, fmt.Errorf("invalid GitHub %s reference: %s", sourceRef.Kind, sourceRef.Reference)
	}

	return workitem.WorkItem{
		Source:      sourceRef,
		Identifier:  identifier,
		Title:       strings.TrimSpace(payload.Title),
		Description: strings.TrimSpace(payload.Body),
		URL:         strings.TrimSpace(payload.URL),
	}, nil
}
