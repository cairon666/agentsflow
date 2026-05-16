package install

import "testing"

func TestFormatSummaryIncludesConflictFiles(t *testing.T) {
	summary := FormatSummary(Plan{
		Target: "claude",
		Scope:  "project",
		Actions: []Action{
			{Path: "managed.md", Kind: ActionUpdate},
			{Path: ".claude/settings.json", Kind: ActionConflict},
		},
	})
	want := "Create: 0\nUpdate: 1\nSkip: 0\nConflicts: 1\n\nConflict files:\n- .claude/settings.json\n"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

func TestFormatSummaryOmitsConflictFilesWhenPlanHasNoConflicts(t *testing.T) {
	summary := FormatSummary(Plan{
		Target: "codex",
		Scope:  "project",
		Actions: []Action{
			{Path: "AGENTS.md", Kind: ActionCreate},
			{Path: ".codex/config.toml", Kind: ActionSkip},
		},
	})
	want := "Create: 1\nUpdate: 0\nSkip: 1\nConflicts: 0\n"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}
