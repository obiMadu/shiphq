package jira

import (
	"fmt"
	"strings"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type ticketProvider struct{}

func init() {
	source.Register("jira", "ticket", ticketProvider{})
}

func (ticketProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	identifier := workitem.NormalizeIdentifier(sourceRef.Reference)
	if identifier == "" {
		return workitem.WorkItem{}, fmt.Errorf("invalid Jira ticket reference: %s", sourceRef.Reference)
	}

	return workitem.WorkItem{
		Mode:       workitem.ModeImplement,
		Source:     sourceRef,
		Identifier: identifier,
		Title:      strings.TrimSpace(sourceRef.Reference),
	}, fmt.Errorf("Jira adapter not yet implemented (ticket: %s)", sourceRef.Reference)
}
