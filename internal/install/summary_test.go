package install

import (
	"strings"
	"testing"
)

func TestFormatDryRunFilePreviewIncludesContentAndMergeDiff(t *testing.T) {
	preview := FormatDryRunFilePreview(Plan{
		Target: "codex",
		Scope:  "project",
		Actions: []Action{
			{Path: ".codex/agents", Kind: ActionCleanDir},
			{Path: "AGENTS.md", Kind: ActionCreate, Content: []byte("# Test\n"), Strategy: StrategyOverwrite},
			{
				Path:            ".codex/config.toml",
				Kind:            ActionUpdate,
				Content:         []byte("model = 'new'\ncustom = true\n"),
				ExistingContent: []byte("model = 'old'\ncustom = true\n"),
				Strategy:        StrategyMerge,
			},
			{Path: "managed.md", Kind: ActionOverwrite, Content: []byte("managed\n"), Strategy: StrategyOverwrite},
			{Path: "same.md", Kind: ActionSkip},
			{Path: "manual.md", Kind: ActionConflict},
		},
	})
	for _, want := range []string{
		"Files:",
		"+ AGENTS.md (create)\n--- planned content ---\n# Test\n--- end planned content ---",
		"+/- .codex/config.toml (update)\n--- planned content ---\nmodel = 'new'\ncustom = true\n--- end planned content ---",
		"+/- managed.md (overwrite)\n--- planned content ---\nmanaged\n--- end planned content ---",
		"= same.md (skip)\n--- planned content ---\n--- end planned content ---",
		"! manual.md (conflict)",
		"--- merge diff ---",
		"--- .codex/config.toml (current)",
		"+++ .codex/config.toml (planned)",
		"-model = 'old'",
		"+model = 'new'",
	} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q:\n%s", want, preview)
		}
	}
	for _, noise := range []string{
		"Dry run install plan:",
		"Clean directories:",
		"Create files:",
		"Update files:",
		"Skip files:",
		"Merge diffs:",
	} {
		if strings.Contains(preview, noise) {
			t.Fatalf("preview contains noise %q:\n%s", noise, preview)
		}
	}
}
