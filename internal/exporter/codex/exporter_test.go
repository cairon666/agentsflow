package codex

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

func TestExportCodexProjectConfig(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, "AGENTS.md"), "# Shared\n")
	writeFile(t, filepath.Join(workDir, ".codex", "agents", "reviewer.toml"), `
name = 'Review Bot'
description = 'Reviews code.'
developer_instructions = '''
Review carefully.
'''
model = 'gpt-review'
model_reasoning_effort = 'high'
sandbox_mode = 'workspace-write'
approval_policy = 'on-request'
web_search = 'live'

[sandbox_workspace_write]
network_access = true
`)

	result, err := (Exporter{}).Export(context.Background(), exportInput(workDir))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.HasErrors(flowmodel.ValidateSpec(result.Spec)) {
		t.Fatalf("exported spec is invalid: %#v", result.Spec)
	}
	agent := result.Spec.Agents["review-bot"]
	if agent.Description != "Reviews code." {
		t.Fatalf("description = %q", agent.Description)
	}
	if agent.PermissionProfile != "review-bot-permissions" {
		t.Fatalf("permission profile = %q", agent.PermissionProfile)
	}
	if agent.ModelSlot != "review-bot-model" {
		t.Fatalf("model slot = %q", agent.ModelSlot)
	}
	if result.Spec.ModelSlots[agent.ModelSlot].Fallback != "" {
		t.Fatalf("agent model slot fallback = %q", result.Spec.ModelSlots[agent.ModelSlot].Fallback)
	}
	caps := result.Spec.PermissionProfiles[agent.PermissionProfile].Capabilities
	for capability, want := range map[string]string{
		"edit_files": "allow",
		"run_shell":  "ask",
		"fetch_urls": "allow",
		"web_search": "allow",
	} {
		if caps[capability] != want {
			t.Fatalf("%s = %q, want %q in %#v", capability, caps[capability], want, caps)
		}
	}
	if result.Spec.Instructions["AGENTS.md"] != "# Shared\n" {
		t.Fatalf("instructions = %q", result.Spec.Instructions["AGENTS.md"])
	}
}

func TestExportCodexWarnsWhenInstructionsMissing(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, ".codex", "agents", "reviewer.toml"), `
name = 'reviewer'
description = 'Reviews code.'
developer_instructions = 'Review.'
`)

	result, err := (Exporter{}).Export(context.Background(), exportInput(workDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected missing instructions warning")
	}
}

func TestExportCodexCreatesPermissionProfileForEachAgent(t *testing.T) {
	workDir := t.TempDir()
	for _, name := range []string{"one", "two", "three"} {
		writeFile(t, filepath.Join(workDir, ".codex", "agents", name+".toml"), `
name = '`+name+`'
description = 'Agent `+name+`.'
developer_instructions = 'Work.'
`)
	}

	result, err := (Exporter{}).Export(context.Background(), exportInput(workDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Spec.ModelSlots) != 4 {
		t.Fatalf("model slot count = %d, want main plus one per agent: %#v", len(result.Spec.ModelSlots), result.Spec.ModelSlots)
	}
	for _, agentID := range []string{"one", "two", "three"} {
		agent := result.Spec.Agents[agentID]
		if agent.ModelSlot != agentID+"-model" {
			t.Fatalf("%s model slot = %q", agentID, agent.ModelSlot)
		}
		if agent.PermissionProfile != agentID+"-permissions" {
			t.Fatalf("%s permission profile = %q", agentID, agent.PermissionProfile)
		}
	}
	if len(result.Spec.PermissionProfiles) != 3 {
		t.Fatalf("permission profile count = %d, want one per agent: %#v", len(result.Spec.PermissionProfiles), result.Spec.PermissionProfiles)
	}
}

func TestExportCodexFailsWhenAgentsMissing(t *testing.T) {
	_, err := (Exporter{}).Export(context.Background(), exportInput(t.TempDir()))
	if err == nil {
		t.Fatal("expected missing agents error")
	}
}

func exportInput(workDir string) exportcore.ExportInput {
	return exportcore.ExportInput{
		Source:  binding.TargetCodex,
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
