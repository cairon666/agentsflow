package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/adapter/codex"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/builder"
	"github.com/cairon666/agentsflow/internal/install"
)

func TestUseWritesCodexFilesWithFakePrompter(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	templatePath := filepath.Join(workDir, "template.yaml")
	if err := os.WriteFile(templatePath, []byte(testTemplate), 0o644); err != nil {
		t.Fatal(err)
	}
	application := App{
		Registry: adapter.NewRegistry(codex.Adapter{}),
		Writer:   install.NewWriter(),
		Stdout:   &bytes.Buffer{},
		WorkDir:  workDir,
		HomeDir:  homeDir,
	}
	if err := application.Use(t.Context(), templatePath, fakePrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".codex", "agents", "reviewer.toml")); err != nil {
		t.Fatalf("expected codex agent file: %v", err)
	}
}

type fakePrompter struct{}

func (fakePrompter) ChooseTarget([]builder.TargetOption) (binding.Target, error) {
	return binding.TargetCodex, nil
}

func (fakePrompter) AskModel(string, string) (string, error) {
	return "gpt-test", nil
}

func (fakePrompter) ChooseScope() (binding.Scope, error) {
	return binding.ScopeProject, nil
}

func (fakePrompter) Confirm(string) (bool, error) {
	return true, nil
}

const testTemplate = `
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
