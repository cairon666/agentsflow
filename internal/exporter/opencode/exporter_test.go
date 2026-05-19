package opencode

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	exportcore "github.com/cairon666/agentsflow/internal/exporter"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

func TestExportOpenCodeProjectConfig(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, "AGENTS.md"), "# OpenCode shared\n")
	writeFile(t, filepath.Join(workDir, ".opencode", "agents", "implementer.md"), `---
description: Implements changes.
reasoningEffort: high
permission:
  edit: allow
  bash: ask
  webfetch: allow
  websearch: deny
  task: allow
---

Implement the task.
`)

	result, err := (Exporter{}).Export(context.Background(), exportInput(workDir))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.HasErrors(flowmodel.ValidateSpec(result.Spec)) {
		t.Fatalf("exported spec is invalid: %#v", result.Spec)
	}
	agent := result.Spec.Agents["implementer"]
	if agent.ModelSlot != "implementer-model" {
		t.Fatalf("model slot = %q", agent.ModelSlot)
	}
	if result.Spec.ModelSlots[agent.ModelSlot].Fallback != "" {
		t.Fatalf("agent model slot fallback = %q", result.Spec.ModelSlots[agent.ModelSlot].Fallback)
	}
	if agent.PermissionProfile != "implementer-permissions" {
		t.Fatalf("permission profile = %q", agent.PermissionProfile)
	}
	caps := result.Spec.PermissionProfiles[agent.PermissionProfile].Capabilities
	for capability, want := range map[string]string{
		"edit_files":   "allow",
		"run_shell":    "ask",
		"fetch_urls":   "allow",
		"web_search":   "deny",
		"spawn_agents": "allow",
	} {
		if caps[capability] != want {
			t.Fatalf("%s = %q, want %q in %#v", capability, caps[capability], want, caps)
		}
	}
}

func TestExportOpenCodeGlobalConfig(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".config", "opencode")
	writeFile(t, filepath.Join(root, "agents", "reviewer.md"), `---
description: Reviews changes.
---

Review the task.
`)

	input := exportcore.ExportInput{
		Source:  binding.TargetOpenCode,
		Scope:   binding.ScopeGlobal,
		WorkDir: t.TempDir(),
		HomeDir: homeDir,
	}
	result, err := (Exporter{}).Export(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Spec.Agents["reviewer"]; !ok {
		t.Fatalf("reviewer agent was not exported: %#v", result.Spec.Agents)
	}
}

func exportInput(workDir string) exportcore.ExportInput {
	return exportcore.ExportInput{
		Source:  binding.TargetOpenCode,
		Scope:   binding.ScopeProject,
		WorkDir: workDir,
		HomeDir: workDir,
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
