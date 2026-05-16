package target

import (
	"context"
	"testing"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/install"
)

func TestRegistryRegistersFakeTargetFromMetadata(t *testing.T) {
	renderer := fakeRenderer{
		metadata: Metadata{
			Name:    binding.Target("fake"),
			Aliases: []string{"fake-cli", "f"},
			Scopes:  []binding.Scope{binding.ScopeProject},
		},
	}
	registry := NewRegistry(renderer)

	resolved, err := registry.Resolve("fake-cli")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "fake" {
		t.Fatalf("resolved target = %q, want fake", resolved)
	}

	got, err := registry.Get("f")
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata().Name != "fake" {
		t.Fatalf("renderer target = %q, want fake", got.Metadata().Name)
	}
}

func TestValidateSupportedScopeRejectsUnsupportedScope(t *testing.T) {
	diags := ValidateSupportedScope(Metadata{
		Name:   binding.Target("fake"),
		Scopes: []binding.Scope{binding.ScopeProject},
	}, binding.ScopeGlobal)

	if !diagnostic.HasErrors(diags) {
		t.Fatalf("diagnostics = %v, want unsupported scope error", diags)
	}
}

type fakeRenderer struct {
	metadata Metadata
}

func (r fakeRenderer) Metadata() Metadata {
	return r.metadata
}

func (r fakeRenderer) Validate(context.Context, RenderInput) []diagnostic.Diagnostic {
	return nil
}

func (r fakeRenderer) Render(context.Context, RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic) {
	return install.ArtifactSet{Target: string(r.metadata.Name), Scope: string(binding.ScopeProject)}, nil
}
