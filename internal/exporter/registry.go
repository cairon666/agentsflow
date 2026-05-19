package exporter

import (
	"fmt"
	"strings"

	"github.com/cairon666/agentsflow/internal/binding"
)

// Registry stores source exporters by canonical name and alias.
type Registry struct {
	exporters map[binding.Target]Exporter
	aliases   map[string]binding.Target
}

// NewRegistry creates a registry for source exporters.
func NewRegistry(exporters ...Exporter) Registry {
	registry := Registry{
		exporters: make(map[binding.Target]Exporter, len(exporters)),
		aliases:   make(map[string]binding.Target),
	}
	for _, item := range exporters {
		metadata := item.Metadata()
		registry.exporters[metadata.Name] = item
		registry.aliases[strings.ToLower(string(metadata.Name))] = metadata.Name
		for _, alias := range metadata.Aliases {
			registry.aliases[strings.ToLower(alias)] = metadata.Name
		}
	}
	return registry
}

// Get resolves a source or alias to an exporter.
func (r Registry) Get(name string) (Exporter, error) {
	sourceName, err := r.Resolve(name)
	if err != nil {
		return nil, err
	}
	exporter, ok := r.exporters[sourceName]
	if !ok {
		return nil, fmt.Errorf("source %q is not registered", sourceName)
	}
	return exporter, nil
}

// Resolve returns the canonical source for a source name or alias.
func (r Registry) Resolve(name string) (binding.Target, error) {
	source, ok := r.aliases[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", fmt.Errorf("unknown source %q", name)
	}
	return source, nil
}

// All returns all registered exporters.
func (r Registry) All() []Exporter {
	out := make([]Exporter, 0, len(r.exporters))
	for _, exporter := range r.exporters {
		out = append(out, exporter)
	}
	return out
}
