package claude

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

func TestExportClaudeProjectConfig(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, "CLAUDE.md"), "# Claude shared\n")
	writeFile(t, filepath.Join(workDir, ".claude", "agents", "scout.md"), `---
name: Scout Agent
description: Reads docs.
effort: low
permissionMode: plan
tools:
  - Read
  - WebFetch
  - WebSearch
  - Task
disallowedTools:
  - Edit
  - Write
---

Scout the repo.
`)

	result, err := (Exporter{}).Export(context.Background(), exportInput(workDir))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.HasErrors(flowmodel.ValidateSpec(result.Spec)) {
		t.Fatalf("exported spec is invalid: %#v", result.Spec)
	}
	agent := result.Spec.Agents["scout-agent"]
	if agent.ModelSlot != "scout-agent-model" {
		t.Fatalf("model slot = %q", agent.ModelSlot)
	}
	if result.Spec.ModelSlots[agent.ModelSlot].Fallback != "" {
		t.Fatalf("agent model slot fallback = %q", result.Spec.ModelSlots[agent.ModelSlot].Fallback)
	}
	if agent.PermissionProfile != "scout-agent-permissions" {
		t.Fatalf("permission profile = %q", agent.PermissionProfile)
	}
	caps := result.Spec.PermissionProfiles[agent.PermissionProfile].Capabilities
	for capability, want := range map[string]string{
		"edit_files":   "deny",
		"run_shell":    "deny",
		"fetch_urls":   "allow",
		"web_search":   "allow",
		"spawn_agents": "allow",
	} {
		if caps[capability] != want {
			t.Fatalf("%s = %q, want %q in %#v", capability, caps[capability], want, caps)
		}
	}
	if result.Spec.Instructions["AGENTS.md"] != "# Claude shared\n" {
		t.Fatalf("instructions = %q", result.Spec.Instructions["AGENTS.md"])
	}
}

func TestExportClaudeFailsOnEmptyPrompt(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, ".claude", "agents", "empty.md"), `---
name: empty
description: Empty prompt.
---
`)

	_, err := (Exporter{}).Export(context.Background(), exportInput(workDir))
	if err == nil {
		t.Fatal("expected empty prompt error")
	}
}

func exportInput(workDir string) exportcore.ExportInput {
	return exportcore.ExportInput{
		Source:  binding.TargetClaude,
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
