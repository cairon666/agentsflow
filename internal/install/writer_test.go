package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriterCleansDirectoryRecursivelyBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".codex", "agents")
	staleNested := filepath.Join(agentsDir, "nested", "stale.toml")
	if err := os.MkdirAll(filepath.Dir(staleNested), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleNested, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	newAgent := filepath.Join(agentsDir, "reviewer.toml")
	plan := Plan{
		Actions: []Action{
			{Path: agentsDir, Kind: ActionCleanDir},
			{Path: newAgent, Kind: ActionCreate, Content: []byte("new")},
		},
	}

	if err := (Writer{}).Apply(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staleNested); !os.IsNotExist(err) {
		t.Fatalf("stale nested file still exists or stat failed unexpectedly: %v", err)
	}
	content, err := os.ReadFile(newAgent)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("new agent content = %q", content)
	}
}

func TestWriterCleanMissingDirectoryIsNoop(t *testing.T) {
	dir := t.TempDir()
	plan := Plan{Actions: []Action{{Path: filepath.Join(dir, "missing"), Kind: ActionCleanDir}}}

	if err := (Writer{}).Apply(plan); err != nil {
		t.Fatal(err)
	}
}

func TestWriterRejectsParentCleanDirectory(t *testing.T) {
	plan := Plan{Actions: []Action{{Path: "..", Kind: ActionCleanDir}}}

	err := (Writer{}).Apply(plan)
	if err == nil {
		t.Fatal("expected unsafe directory error")
	}
	if !strings.Contains(err.Error(), "refusing to clean unsafe directory") {
		t.Fatalf("error = %q", err)
	}
}

func TestWriterRejectsSymlinkCleanDirectory(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "agents")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	plan := Plan{Actions: []Action{{Path: linkPath, Kind: ActionCleanDir}}}

	err := (Writer{}).Apply(plan)
	if err == nil {
		t.Fatal("expected symlink directory error")
	}
	if !strings.Contains(err.Error(), "refusing to clean symlink directory") {
		t.Fatalf("error = %q", err)
	}
}

func TestWriterOverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Actions: []Action{{Path: path, Kind: ActionOverwrite, Content: []byte("new")}}}

	if err := (Writer{}).Apply(plan); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("content = %q", content)
	}
}
