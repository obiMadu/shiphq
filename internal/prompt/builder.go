package prompt

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/obiMadu/wtmag/internal/repository"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type BuildOptions struct {
	OverrideText string
	PRFlag       bool
}

type FrameData struct {
	SourceLabel   string
	Context       string
	OverrideText  string
	PRInstruction string
}

type PRData struct {
	Reference string
}

func Build(workItem workitem.WorkItem, repoTarget repository.Target, opts BuildOptions) (string, error) {
	if opts.PRFlag && workItem.Mode == workitem.ModeReview {
		return "", fmt.Errorf("--pr is only valid with implementation tasks; review tasks already inspect a PR")
	}

	sourceLabel := describeWorkItem(workItem)
	context := workItem.Context()
	overrideText := strings.TrimSpace(opts.OverrideText)
	host := string(repoTarget.Host)

	var prInstruction string
	if opts.PRFlag && workItem.Mode == workitem.ModeImplement {
		reference := resolveReference(workItem, repoTarget)
		prTemplate, err := resolveTemplate("pr", host)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := prTemplate.Execute(&buf, PRData{Reference: reference}); err != nil {
			return "", fmt.Errorf("failed to render pr template: %w", err)
		}
		prInstruction = strings.TrimSpace(buf.String())
	}

	var templateType string
	switch workItem.Mode {
	case workitem.ModeImplement:
		templateType = "implement"
	case workitem.ModeReview:
		templateType = "review"
	default:
		return context, nil
	}

	frameTemplate, err := resolveTemplate(templateType, host)
	if err != nil {
		return "", err
	}

	data := FrameData{
		SourceLabel:   sourceLabel,
		Context:       context,
		OverrideText:  overrideText,
		PRInstruction: prInstruction,
	}

	var buf bytes.Buffer
	if err := frameTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render %s template: %w", templateType, err)
	}

	return strings.TrimSpace(buf.String()), nil
}

func describeWorkItem(workItem workitem.WorkItem) string {
	switch {
	case workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt":
		return ""
	case workItem.Source.System == "github" && workItem.Source.Kind == "issue":
		return fmt.Sprintf("GitHub issue #%s", workItem.Source.Reference)
	case workItem.Source.System == "github" && workItem.Source.Kind == "pr":
		return fmt.Sprintf("GitHub PR #%s", workItem.Source.Reference)
	default:
		genericLabel := strings.TrimSpace(workItem.Source.System + " " + workItem.Source.Kind)
		if workItem.Source.Reference == "" {
			return genericLabel
		}
		return strings.TrimSpace(genericLabel + " " + workItem.Source.Reference)
	}
}

func resolveReference(workItem workitem.WorkItem, repoTarget repository.Target) string {
	sourceReference := strings.TrimSpace(workItem.Source.Reference)
	if sourceReference == "" {
		return ""
	}

	if workItem.Source.System == "github" && workItem.Source.Kind == "issue" && repoTarget.Host == repository.HostGitHub {
		return fmt.Sprintf(", make sure the PR body references GitHub issue #%s with a non-closing reference like `Refs #%s` so the issue gets a backlink without being closed", sourceReference, sourceReference)
	}

	return ""
}
