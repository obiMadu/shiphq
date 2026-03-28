package source

import (
	"fmt"

	"github.com/obiMadu/shiphq/internal/workitem"
)

type Provider interface {
	Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error)
}

var providers = map[string]Provider{}

func Register(system, kind string, provider Provider) {
	registryKey := workitem.SourceRef{System: system, Kind: kind}.RegistryKey()
	if _, exists := providers[registryKey]; exists {
		panic(fmt.Sprintf("provider already registered for %s", registryKey))
	}

	providers[registryKey] = provider
}

func Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	provider, exists := providers[sourceRef.RegistryKey()]
	if !exists {
		return workitem.WorkItem{}, fmt.Errorf("unsupported source: %s %s", sourceRef.System, sourceRef.Kind)
	}

	return provider.Fetch(sourceRef)
}
