package exporter

import (
	"context"
	"testing"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
)

func TestRegistryRegistersFakeSourceFromMetadata(t *testing.T) {
	source := fakeExporter{
		metadata: Metadata{
			Name:    binding.Target("fake"),
			Aliases: []string{"fake-cli", "f"},
			Scopes:  []binding.Scope{binding.ScopeProject},
		},
	}
	registry := NewRegistry(source)

	resolved, err := registry.Resolve("fake-cli")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "fake" {
		t.Fatalf("resolved source = %q, want fake", resolved)
	}

	got, err := registry.Get("f")
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata().Name != "fake" {
		t.Fatalf("exporter source = %q, want fake", got.Metadata().Name)
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

type fakeExporter struct {
	metadata Metadata
}

func (e fakeExporter) Metadata() Metadata {
	return e.metadata
}

func (e fakeExporter) Export(context.Context, ExportInput) (ExportResult, error) {
	return ExportResult{}, nil
}
