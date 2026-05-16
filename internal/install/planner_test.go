package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPlanClassifiesCreateSkipConflict(t *testing.T) {
	dir := t.TempDir()
	createPath := filepath.Join(dir, "create.txt")
	skipPath := filepath.Join(dir, "skip.txt")
	conflictPath := filepath.Join(dir, "conflict.txt")
	if err := os.WriteFile(skipPath, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conflictPath, []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := BuildPlan(ArtifactSet{
		Target: "test",
		Scope:  "project",
		Files: []DesiredFile{
			{Path: createPath, Content: []byte("new")},
			{Path: skipPath, Content: []byte("same")},
			{Path: conflictPath, Content: []byte("generated")},
		},
	})
	kinds := map[string]ActionKind{}
	for _, action := range plan.Actions {
		kinds[action.Path] = action.Kind
	}
	if kinds[createPath] != ActionCreate {
		t.Fatalf("create path kind = %q", kinds[createPath])
	}
	if kinds[skipPath] != ActionSkip {
		t.Fatalf("skip path kind = %q", kinds[skipPath])
	}
	if kinds[conflictPath] != ActionConflict {
		t.Fatalf("conflict path kind = %q", kinds[conflictPath])
	}
}

func TestBuildPlanMergeStrategyUpdatesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"model":"old"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := BuildPlan(ArtifactSet{
		Target: "claude",
		Scope:  "project",
		Files: []DesiredFile{
			{Path: path, Content: []byte(`{"model":"new"}`), Strategy: StrategyMerge},
		},
	})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d", len(plan.Actions))
	}
	if plan.Actions[0].Kind != ActionUpdate {
		t.Fatalf("kind = %q, want update", plan.Actions[0].Kind)
	}
	if plan.Actions[0].Strategy != StrategyMerge {
		t.Fatalf("strategy = %q, want merge", plan.Actions[0].Strategy)
	}
	if string(plan.Actions[0].ExistingContent) != `{"model":"old"}` {
		t.Fatalf("existing content = %q, want old file content", plan.Actions[0].ExistingContent)
	}
}

func TestBuildPlanOverwriteStrategyOverwritesExistingDifferentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := BuildPlan(ArtifactSet{
		Target: "codex",
		Scope:  "project",
		Files: []DesiredFile{
			{Path: path, Content: []byte("generated"), Strategy: StrategyOverwrite},
		},
	})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d", len(plan.Actions))
	}
	if plan.Actions[0].Kind != ActionOverwrite {
		t.Fatalf("kind = %q, want overwrite", plan.Actions[0].Kind)
	}
	if plan.Actions[0].Strategy != StrategyOverwrite {
		t.Fatalf("strategy = %q, want overwrite", plan.Actions[0].Strategy)
	}
}

func TestBuildPlanCleanDirPrecedesAgentWrites(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".codex", "agents")
	agentPath := filepath.Join(agentsDir, "reviewer.toml")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentPath, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := BuildPlan(ArtifactSet{
		Target:    "codex",
		Scope:     "project",
		CleanDirs: []string{agentsDir},
		Files: []DesiredFile{
			{Path: agentPath, Content: []byte("same"), Strategy: StrategyOverwrite},
		},
	})
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %d", len(plan.Actions))
	}
	if plan.Actions[0].Kind != ActionCleanDir || plan.Actions[0].Path != agentsDir {
		t.Fatalf("first action = %#v, want clean dir for %s", plan.Actions[0], agentsDir)
	}
	if plan.Actions[1].Kind != ActionCreate {
		t.Fatalf("agent action kind = %q, want create after cleanup", plan.Actions[1].Kind)
	}
}

func TestBuildPlanCreateOnlyStrategyConflictsOnExistingDifferentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := BuildPlan(ArtifactSet{
		Target: "test",
		Scope:  "project",
		Files: []DesiredFile{
			{Path: path, Content: []byte("generated"), Strategy: StrategyCreateOnly},
		},
	})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d", len(plan.Actions))
	}
	if plan.Actions[0].Kind != ActionConflict {
		t.Fatalf("kind = %q, want conflict", plan.Actions[0].Kind)
	}
}
