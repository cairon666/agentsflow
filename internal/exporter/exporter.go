package exporter

import (
	"context"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

// Metadata describes a native source exporter for registration and selection.
type Metadata struct {
	Name    binding.Target
	Aliases []string
	Scopes  []binding.Scope
}

// ExportInput identifies the native config source to export.
type ExportInput struct {
	Source  binding.Target
	Scope   binding.Scope
	WorkDir string
	HomeDir string
}

// ExportResult contains the exported template and non-blocking diagnostics.
type ExportResult struct {
	Spec        flowmodel.Spec
	Diagnostics []diagnostic.Diagnostic
}

// Exporter reads native agent CLI configuration and converts it to a template spec.
type Exporter interface {
	Metadata() Metadata
	Export(context.Context, ExportInput) (ExportResult, error)
}

// ProjectAndGlobalScopes returns the scopes supported by native sources with local and global config.
func ProjectAndGlobalScopes() []binding.Scope {
	return []binding.Scope{binding.ScopeProject, binding.ScopeGlobal}
}

// ValidateSupportedScope reports whether a selected scope is supported by an exporter.
func ValidateSupportedScope(metadata Metadata, scope binding.Scope) []diagnostic.Diagnostic {
	for _, supported := range metadata.Scopes {
		if supported == scope {
			return nil
		}
	}
	return []diagnostic.Diagnostic{diagnostic.Errorf("source %q does not support %q scope", metadata.Name, scope)}
}
