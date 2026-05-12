package github

import (
	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type pullRequestProvider struct{}

func init() {
	source.Register("github", "pr", pullRequestProvider{})
}

func (pullRequestProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	return fetchWithGitHubCLI("pr", sourceRef, workitem.ModeReview)
}
