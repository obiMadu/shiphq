package github

import (
	"github.com/obiMadu/shiphq/internal/source"
	"github.com/obiMadu/shiphq/internal/workitem"
)

type pullRequestProvider struct{}

func init() {
	source.Register("github", "pr", pullRequestProvider{})
}

func (pullRequestProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	return fetchWithGitHubCLI("pr", sourceRef, workitem.ModeReview)
}
