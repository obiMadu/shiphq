package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/obiMadu/wtmag/internal/workitem"
)

type CreateInput struct {
	SourceRef      workitem.SourceRef
	PromptOverride string
}

func ResolveCreateInput(githubNumber int, jiraTicketID, promptText, typeFlag string) (CreateInput, error) {
	trimmedJiraTicketID := strings.TrimSpace(jiraTicketID)
	trimmedPromptText := strings.TrimSpace(promptText)
	trimmedType := strings.TrimSpace(typeFlag)

	hasGitHubSource := githubNumber != 0
	hasJiraSource := trimmedJiraTicketID != ""
	hasPromptSource := trimmedPromptText != "" && !hasGitHubSource && !hasJiraSource

	sourceCount := 0
	if hasGitHubSource {
		sourceCount++
	}
	if hasJiraSource {
		sourceCount++
	}
	if hasPromptSource {
		sourceCount++
	}

	if sourceCount == 0 {
		return CreateInput{}, fmt.Errorf("must specify --github, --jira, or --prompt")
	}
	if sourceCount > 1 {
		return CreateInput{}, fmt.Errorf("must specify only one source: --github, --jira, or --prompt")
	}

	if hasGitHubSource {
		if githubNumber < 0 {
			return CreateInput{}, fmt.Errorf("--github must be a positive number")
		}
		if trimmedType == "" {
			return CreateInput{}, fmt.Errorf("--type (-t) is required for GitHub (use 'issue' or 'pr')")
		}
		if trimmedType != "issue" && trimmedType != "pr" {
			return CreateInput{}, fmt.Errorf("unknown type '%s' for GitHub (use 'issue' or 'pr')", trimmedType)
		}

		return CreateInput{
			SourceRef: workitem.SourceRef{
				System:    "github",
				Kind:      trimmedType,
				Reference: strconv.Itoa(githubNumber),
			},
			PromptOverride: trimmedPromptText,
		}, nil
	}

	if hasJiraSource {
		if trimmedType != "" {
			return CreateInput{}, fmt.Errorf("--type is not used with --jira")
		}

		return CreateInput{
			SourceRef: workitem.SourceRef{
				System:    "jira",
				Kind:      "ticket",
				Reference: trimmedJiraTicketID,
			},
			PromptOverride: trimmedPromptText,
		}, nil
	}

	if trimmedType != "" {
		return CreateInput{}, fmt.Errorf("--type can only be used with --github")
	}

	return CreateInput{
		SourceRef: workitem.SourceRef{
			System:    "prompt",
			Kind:      "prompt",
			Reference: trimmedPromptText,
		},
	}, nil
}
