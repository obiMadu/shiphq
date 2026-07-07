package github

import (
	"fmt"
	"strings"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type pullRequestProvider struct{}

func init() {
	source.Register("github", "pr", workitem.ModeReview, pullRequestProvider{})
}

func (pullRequestProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	if strings.TrimSpace(sourceRef.Reference) == "" {
		return workitem.WorkItem{}, fmt.Errorf("GitHub %s reference cannot be empty", sourceRef.Kind)
	}

	payload, err := runView(sourceRef, []string{"pr", "view", sourceRef.Reference, "--json", "title,body,url,headRefName"})
	if err != nil {
		return workitem.WorkItem{}, err
	}

	item, err := baseWorkItem(sourceRef, payload)
	if err != nil {
		return workitem.WorkItem{}, err
	}
	item.TargetBranch = strings.TrimSpace(payload.HeadRefName)
	return item, nil
}
