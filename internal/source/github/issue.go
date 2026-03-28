package github

import (
	"github.com/obiMadu/shiphq/internal/source"
	"github.com/obiMadu/shiphq/internal/workitem"
)

type issueProvider struct{}

func init() {
	source.Register("github", "issue", issueProvider{})
}

func (issueProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	return fetchWithGitHubCLI("issue", sourceRef, workitem.ModeImplement)
}
