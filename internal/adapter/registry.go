package adapter

import (
	"fmt"
	"strings"

	"github.com/cairon666/agentflow/internal/binding"
)

// Registry stores target adapters by canonical name and alias.
type Registry struct {
	adapters map[binding.Target]Adapter
	aliases  map[string]binding.Target
}

// NewRegistry creates a registry for adapters.
func NewRegistry(adapters ...Adapter) Registry {
	registry := Registry{
		adapters: make(map[binding.Target]Adapter, len(adapters)),
		aliases:  make(map[string]binding.Target),
	}
	for _, item := range adapters {
		target := item.Target()
		registry.adapters[target] = item
		registry.aliases[string(target)] = target
		for _, alias := range item.Aliases() {
			registry.aliases[strings.ToLower(alias)] = target
		}
	}
	return registry
}

// Get resolves a target or alias to an adapter.
func (r Registry) Get(name string) (Adapter, error) {
	target, err := r.Resolve(name)
	if err != nil {
		return nil, err
	}
	adapter, ok := r.adapters[target]
	if !ok {
		return nil, fmt.Errorf("target %q is not registered", target)
	}
	return adapter, nil
}

// Resolve returns the canonical target for a target name or alias.
func (r Registry) Resolve(name string) (binding.Target, error) {
	target, ok := r.aliases[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", fmt.Errorf("unknown target %q", name)
	}
	return target, nil
}

// All returns all registered adapters.
func (r Registry) All() []Adapter {
	out := make([]Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		out = append(out, adapter)
	}
	return out
}
