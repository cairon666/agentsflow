package install

import "testing"

func TestFormatSummaryIncludesConflictFiles(t *testing.T) {
	summary := FormatSummary(Plan{
		Target: "claude",
		Scope:  "project",
		Actions: []Action{
			{Path: ".claude/agents", Kind: ActionCleanDir},
			{Path: "managed.md", Kind: ActionUpdate},
			{Path: "CLAUDE.md", Kind: ActionOverwrite},
			{Path: ".claude/settings.json", Kind: ActionConflict},
		},
	})
	want := "Clean: 1\nCreate: 0\nUpdate: 1\nOverwrite: 1\nSkip: 0\nConflicts: 1\n\nOverwrite files:\n- CLAUDE.md\n\nConflict files:\n- .claude/settings.json\n"
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
	want := "Clean: 0\nCreate: 1\nUpdate: 0\nOverwrite: 0\nSkip: 1\nConflicts: 0\n"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}
