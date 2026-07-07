package source

import (
	"fmt"
	"sort"
	"strings"

	"github.com/obiMadu/wtmag/internal/workitem"
)

type Provider interface {
	Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error)
}

type Entry struct {
	Kind    string
	Mode    workitem.WorkMode
	Fetcher Provider
}

var providers = map[string]Entry{}

func Register(system, kind string, mode workitem.WorkMode, provider Provider) {
	registryKey := workitem.SourceRef{System: system, Kind: kind}.RegistryKey()
	if _, exists := providers[registryKey]; exists {
		panic(fmt.Sprintf("provider already registered for %s", registryKey))
	}

	providers[registryKey] = Entry{Kind: kind, Mode: mode, Fetcher: provider}
}

func KindsFor(system string) []Entry {
	prefix := system + ":"
	var entries []Entry
	for key, entry := range providers {
		if strings.HasPrefix(key, prefix) {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Kind < entries[j].Kind })
	return entries
}

func Resolve(system, kind string) (Provider, workitem.WorkMode, error) {
	entry, exists := providers[workitem.SourceRef{System: system, Kind: kind}.RegistryKey()]
	if !exists {
		return nil, "", fmt.Errorf("unsupported source: %s %s", system, kind)
	}
	return entry.Fetcher, entry.Mode, nil
}

func Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	entry, exists := providers[sourceRef.RegistryKey()]
	if !exists {
		return workitem.WorkItem{}, fmt.Errorf("unsupported source: %s %s", sourceRef.System, sourceRef.Kind)
	}

	return entry.Fetcher.Fetch(sourceRef)
}
