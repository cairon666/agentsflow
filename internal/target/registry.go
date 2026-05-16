package target

import (
	"fmt"
	"strings"

	"github.com/cairon666/agentsflow/internal/binding"
)

// Registry stores target renderers by canonical name and alias.
type Registry struct {
	renderers map[binding.Target]Renderer
	aliases   map[string]binding.Target
}

// NewRegistry creates a registry for target renderers.
func NewRegistry(renderers ...Renderer) Registry {
	registry := Registry{
		renderers: make(map[binding.Target]Renderer, len(renderers)),
		aliases:   make(map[string]binding.Target),
	}
	for _, item := range renderers {
		metadata := item.Metadata()
		registry.renderers[metadata.Name] = item
		registry.aliases[strings.ToLower(string(metadata.Name))] = metadata.Name
		for _, alias := range metadata.Aliases {
			registry.aliases[strings.ToLower(alias)] = metadata.Name
		}
	}
	return registry
}

// Get resolves a target or alias to a renderer.
func (r Registry) Get(name string) (Renderer, error) {
	targetName, err := r.Resolve(name)
	if err != nil {
		return nil, err
	}
	renderer, ok := r.renderers[targetName]
	if !ok {
		return nil, fmt.Errorf("target %q is not registered", targetName)
	}
	return renderer, nil
}

// Resolve returns the canonical target for a target name or alias.
func (r Registry) Resolve(name string) (binding.Target, error) {
	target, ok := r.aliases[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", fmt.Errorf("unknown target %q", name)
	}
	return target, nil
}

// All returns all registered renderers.
func (r Registry) All() []Renderer {
	out := make([]Renderer, 0, len(r.renderers))
	for _, renderer := range r.renderers {
		out = append(out, renderer)
	}
	return out
}
