package prompt

import (
	"fmt"
	"strings"

	"github.com/obiMadu/shiphq/internal/repository"
	"github.com/obiMadu/shiphq/internal/workitem"
)

func BuildDefault(workItem workitem.WorkItem, repositoryTarget repository.Target) string {
	if workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt" {
		return strings.TrimSpace(workItem.Description)
	}

	workItemContext := workItem.Context()

	switch workItem.Mode {
	case workitem.ModeReview:
		if workItemContext == "" {
			return fmt.Sprintf("Review %s.", describeWorkItem(workItem))
		}
		return fmt.Sprintf("Review %s.\n\n%s", describeWorkItem(workItem), workItemContext)
	case workitem.ModeImplement:
		if workItemContext == "" {
			return fmt.Sprintf("Implement %s.\n\n%s", describeWorkItem(workItem), deliveryInstruction(repositoryTarget))
		}
		return fmt.Sprintf("Implement %s.\n\n%s\n\n%s", describeWorkItem(workItem), workItemContext, deliveryInstruction(repositoryTarget))
	default:
		return workItemContext
	}
}

func BuildOverride(workItem workitem.WorkItem, repositoryTarget repository.Target, instructions string) string {
	trimmedInstructions := strings.TrimSpace(instructions)
	if trimmedInstructions == "" {
		return BuildDefault(workItem, repositoryTarget)
	}

	if workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt" {
		return trimmedInstructions
	}

	workItemContext := workItem.Context()
	if workItemContext == "" {
		return trimmedInstructions
	}

	return fmt.Sprintf("%s\n\nContext:\n%s", trimmedInstructions, workItemContext)
}

func describeWorkItem(workItem workitem.WorkItem) string {
	switch {
	case workItem.Source.System == "github" && workItem.Source.Kind == "issue":
		return fmt.Sprintf("GitHub issue #%s", workItem.Source.Reference)
	case workItem.Source.System == "github" && workItem.Source.Kind == "pr":
		return fmt.Sprintf("GitHub PR #%s", workItem.Source.Reference)
	case workItem.Source.System == "jira" && workItem.Source.Kind == "ticket":
		return fmt.Sprintf("Jira ticket %s", workItem.Source.Reference)
	default:
		genericLabel := strings.TrimSpace(workItem.Source.System + " " + workItem.Source.Kind)
		if workItem.Source.Reference == "" {
			return genericLabel
		}
		return strings.TrimSpace(genericLabel + " " + workItem.Source.Reference)
	}
}

func deliveryInstruction(repositoryTarget repository.Target) string {
	switch repositoryTarget.Host {
	case repository.HostGitHub:
		return "When the implementation is complete, commit your changes, push the branch, open and submit a GitHub PR with gh against the appropriate base branch, and report the PR URL."
	case repository.HostGitLab:
		return "When the implementation is complete, commit your changes, push the branch, open and submit a GitLab merge request with glab against the appropriate base branch, and report the merge request URL."
	case repository.HostBitbucket:
		return "When the implementation is complete, commit your changes, push the branch, open and submit a Bitbucket pull request against the appropriate base branch, and report the pull request URL."
	default:
		return "When the implementation is complete, commit your changes, push the branch, open and submit a pull request against the appropriate base branch, and report the PR URL."
	}
}
