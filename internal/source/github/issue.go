package github

import (
	"fmt"
	"strings"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type issueProvider struct{}

func init() {
	source.Register("github", "issue", workitem.ModeImplement, issueProvider{})
}

func (issueProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	if strings.TrimSpace(sourceRef.Reference) == "" {
		return workitem.WorkItem{}, fmt.Errorf("GitHub %s reference cannot be empty", sourceRef.Kind)
	}

	payload, err := runView(sourceRef, []string{"issue", "view", sourceRef.Reference, "--json", "title,body,url"})
	if err != nil {
		return workitem.WorkItem{}, err
	}

	return baseWorkItem(sourceRef, payload)
}
