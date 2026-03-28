package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/obiMadu/shiphq/internal/workitem"
)

type viewPayload struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
}

func fetchWithGitHubCLI(commandName string, sourceRef workitem.SourceRef, workMode workitem.WorkMode) (workitem.WorkItem, error) {
	if strings.TrimSpace(sourceRef.Reference) == "" {
		return workitem.WorkItem{}, fmt.Errorf("GitHub %s reference cannot be empty", sourceRef.Kind)
	}

	fields := []string{"title", "body", "url"}
	if commandName == "pr" {
		fields = append(fields, "headRefName")
	}

	command := exec.Command("gh", commandName, "view", sourceRef.Reference, "--json", strings.Join(fields, ","))
	output, err := command.CombinedOutput()
	if err != nil {
		return workitem.WorkItem{}, fmt.Errorf("failed to fetch GitHub %s %s: %w\n%s", sourceRef.Kind, sourceRef.Reference, err, output)
	}

	var payload viewPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return workitem.WorkItem{}, fmt.Errorf("failed to decode GitHub %s %s: %w", sourceRef.Kind, sourceRef.Reference, err)
	}

	identifier := workitem.NormalizeIdentifier(sourceRef.Reference)
	if identifier == "" {
		return workitem.WorkItem{}, fmt.Errorf("invalid GitHub %s reference: %s", sourceRef.Kind, sourceRef.Reference)
	}

	return workitem.WorkItem{
		Mode:         workMode,
		Source:       sourceRef,
		Identifier:   identifier,
		TargetBranch: strings.TrimSpace(payload.HeadRefName),
		Title:        strings.TrimSpace(payload.Title),
		Description:  strings.TrimSpace(payload.Body),
		URL:          strings.TrimSpace(payload.URL),
	}, nil
}
