package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/diagnostic"
)

func TestLoadFileReturnsNormalizedFlow(t *testing.T) {
	path := writeTemplate(t, validTemplate)

	result, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %v, want none", result.Diagnostics)
	}
	if result.Flow.ID != "test-flow" {
		t.Fatalf("flow id = %q, want test-flow", result.Flow.ID)
	}
	if result.Flow.ModelSlots["main"].Description != "Main model" {
		t.Fatalf("main model slot was not normalized: %+v", result.Flow.ModelSlots["main"])
	}
	agent := result.Flow.Agents["reviewer"]
	if agent.ID != "reviewer" || agent.ModelSlot != "main" {
		t.Fatalf("agent was not normalized: %+v", agent)
	}
	if result.Flow.Instructions["AGENTS.md"] != "# Test\n" {
		t.Fatalf("instructions = %q, want # Test", result.Flow.Instructions["AGENTS.md"])
	}
}

func TestLoadFileReturnsValidationDiagnostics(t *testing.T) {
	path := writeTemplate(t, invalidTemplate)

	result, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !diagnostic.HasErrors(result.Diagnostics) {
		t.Fatalf("diagnostics = %v, want validation errors", result.Diagnostics)
	}
	if result.Flow.Version != 1 {
		t.Fatalf("flow version = %d, want 1", result.Flow.Version)
	}
}

func TestLoadFileReturnsDecodeError(t *testing.T) {
	path := writeTemplate(t, "id: [")

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "decode template") {
		t.Fatalf("error = %q, want decode template", err)
	}
}

func TestLoadFileReturnsReadError(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read template") {
		t.Fatalf("error = %q, want read template", err)
	}
}

func writeTemplate(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "template.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validTemplate = `
id: test-flow
version: 1
model_slots:
  main:
    description: Main model
permission_profiles:
  read:
    description: Read profile
    capabilities:
      read_files: allow
      edit_files: deny
agents:
  reviewer:
    description: Reviews code
    model_slot: main
    reasoning_effort: medium
    permission_profile: read
    prompt: Review code.
instructions:
  AGENTS.md: |
    # Test
`

const invalidTemplate = `
id: test-flow
version: 1
`
