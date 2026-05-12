package prompt

import (
	"fmt"
	"strings"

	"github.com/obiMadu/wtmag/internal/repository"
	"github.com/obiMadu/wtmag/internal/workitem"
)

func BuildDefault(workItem workitem.WorkItem, repositoryTarget repository.Target) string {
	workItemContext := workItem.Context()

	switch workItem.Mode {
	case workitem.ModeReview:
		if workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt" {
			return appendReviewInstructions(strings.TrimSpace(workItem.Description))
		}
		if workItemContext == "" {
			return fmt.Sprintf("Review %s.\n\n%s", describeWorkItem(workItem), reviewInstruction())
		}
		return fmt.Sprintf("Review %s.\n\n%s\n\n%s", describeWorkItem(workItem), workItemContext, reviewInstruction())
	case workitem.ModeImplement:
		if workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt" {
			return appendImplementationInstructions(strings.TrimSpace(workItem.Description), workItem, repositoryTarget)
		}
		if workItemContext == "" {
			return fmt.Sprintf("Implement %s.\n\n%s", describeWorkItem(workItem), deliveryInstruction(workItem, repositoryTarget))
		}
		return fmt.Sprintf("Implement %s.\n\n%s\n\n%s", describeWorkItem(workItem), workItemContext, deliveryInstruction(workItem, repositoryTarget))
	default:
		return workItemContext
	}
}

func BuildOverride(workItem workitem.WorkItem, repositoryTarget repository.Target, instructions string) string {
	trimmedInstructions := strings.TrimSpace(instructions)
	if trimmedInstructions == "" {
		return BuildDefault(workItem, repositoryTarget)
	}

	if workItem.Mode == workitem.ModeImplement {
		if workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt" {
			return appendImplementationInstructions(trimmedInstructions, workItem, repositoryTarget)
		}

		workItemContext := workItem.Context()
		if workItemContext == "" {
			return appendImplementationInstructions(trimmedInstructions, workItem, repositoryTarget)
		}

		return appendImplementationInstructions(fmt.Sprintf("%s\n\nContext:\n%s", trimmedInstructions, workItemContext), workItem, repositoryTarget)
	}

	if workItem.Mode == workitem.ModeReview {
		if workItem.Source.System == "prompt" && workItem.Source.Kind == "prompt" {
			return appendReviewInstructions(trimmedInstructions)
		}

		workItemContext := workItem.Context()
		if workItemContext == "" {
			return appendReviewInstructions(trimmedInstructions)
		}

		return appendReviewInstructions(fmt.Sprintf("%s\n\nContext:\n%s", trimmedInstructions, workItemContext))
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

func appendImplementationInstructions(promptText string, workItem workitem.WorkItem, repositoryTarget repository.Target) string {
	trimmedPromptText := strings.TrimSpace(promptText)
	if trimmedPromptText == "" {
		return deliveryInstruction(workItem, repositoryTarget)
	}

	return fmt.Sprintf("%s\n\n%s", trimmedPromptText, deliveryInstruction(workItem, repositoryTarget))
}

func appendReviewInstructions(promptText string) string {
	trimmedPromptText := strings.TrimSpace(promptText)
	if trimmedPromptText == "" {
		return reviewInstruction()
	}

	return fmt.Sprintf("%s\n\n%s", trimmedPromptText, reviewInstruction())
}

func deliveryInstruction(workItem workitem.WorkItem, repositoryTarget repository.Target) string {
	requestReferenceInstruction := referenceInstruction(workItem, repositoryTarget)

	switch repositoryTarget.Host {
	case repository.HostGitHub:
		return fmt.Sprintf("When the implementation is complete, commit your changes, push the branch, open and submit a GitHub PR with gh against the appropriate base branch%s, report the PR URL, and stop there. Do not merge, approve, or enable auto-merge on the PR; it will be reviewed separately.", requestReferenceInstruction)
	case repository.HostGitLab:
		return fmt.Sprintf("When the implementation is complete, commit your changes, push the branch, open and submit a GitLab merge request with glab against the appropriate base branch%s, report the merge request URL, and stop there. Do not merge, approve, or enable auto-merge on it; it will be reviewed separately.", requestReferenceInstruction)
	case repository.HostBitbucket:
		return fmt.Sprintf("When the implementation is complete, commit your changes, push the branch, open and submit a Bitbucket pull request against the appropriate base branch%s, report the pull request URL, and stop there. Do not merge, approve, or enable auto-merge on it; it will be reviewed separately.", requestReferenceInstruction)
	default:
		return fmt.Sprintf("When the implementation is complete, commit your changes, push the branch, open and submit a pull request against the appropriate base branch%s, report the PR URL, and stop there. Do not merge, approve, or enable auto-merge on it; it will be reviewed separately.", requestReferenceInstruction)
	}
}

func referenceInstruction(workItem workitem.WorkItem, repositoryTarget repository.Target) string {
	sourceReference := strings.TrimSpace(workItem.Source.Reference)
	if sourceReference == "" {
		return ""
	}

	switch {
	case workItem.Source.System == "github" && workItem.Source.Kind == "issue" && repositoryTarget.Host == repository.HostGitHub:
		return fmt.Sprintf(", make sure the PR body references GitHub issue #%s with a non-closing reference like `Refs #%s` so the issue gets a backlink without being closed", sourceReference, sourceReference)
	default:
		return ""
	}
}

func reviewInstruction() string {
	return "This is a review-only task. Inspect the PR and report your findings back here. Do not edit any files, implement fixes, commit, push, approve, merge, or otherwise modify the PR or branch. If you identify a fix, describe it without making changes."
}
