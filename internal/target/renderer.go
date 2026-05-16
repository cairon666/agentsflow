package target

import (
	"context"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
)

// Metadata describes a target renderer for registration and selection.
type Metadata struct {
	Name    binding.Target
	Aliases []string
	Scopes  []binding.Scope
}

// RenderInput contains a validated flow and user choices.
type RenderInput struct {
	Flow    flowmodel.Flow
	Models  binding.Models
	Scope   binding.Scope
	WorkDir string
	HomeDir string
}

// Renderer renders target-specific desired files.
type Renderer interface {
	Metadata() Metadata
	// Validate returns errors for unsupported input and warnings for lossy target mappings.
	Validate(context.Context, RenderInput) []diagnostic.Diagnostic
	// Render returns desired files and render diagnostics without writing to the filesystem.
	Render(context.Context, RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic)
}

// ProjectAndGlobalScopes returns the scopes supported by targets that can write both local and global config.
func ProjectAndGlobalScopes() []binding.Scope {
	return []binding.Scope{binding.ScopeProject, binding.ScopeGlobal}
}

// ValidateSupportedScope reports whether a selected scope is supported by a target.
func ValidateSupportedScope(metadata Metadata, scope binding.Scope) []diagnostic.Diagnostic {
	for _, supported := range metadata.Scopes {
		if supported == scope {
			return nil
		}
	}
	return []diagnostic.Diagnostic{diagnostic.Errorf("target %q does not support %q scope", metadata.Name, scope)}
}
