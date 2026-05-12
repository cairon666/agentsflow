package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cairon666/agentflow/internal/adapter"
	"github.com/cairon666/agentflow/internal/adapter/codex"
	"github.com/cairon666/agentflow/internal/binding"
	"github.com/cairon666/agentflow/internal/builder"
	"github.com/cairon666/agentflow/internal/install"
)

func TestUseRemoteRepositoryPromptsForSingleTemplate(t *testing.T) {
	workDir := t.TempDir()
	repoDir := t.TempDir()
	writeRemoteTemplate(t, repoDir, "alpha", testTemplate)

	cloner := &fakeGitCloner{sourceDir: repoDir}
	prompter := &remotePrompter{selectedLabel: "alpha"}
	application := remoteAppForTest(workDir, cloner)
	if err := application.Use(t.Context(), "https://example.test/repo.git", prompter); err != nil {
		t.Fatal(err)
	}

	assertTempRepoRemoved(t, cloner.dest)
	if prompter.templateCalls != 1 {
		t.Fatalf("template prompt calls = %d, want 1", prompter.templateCalls)
	}
	if !reflect.DeepEqual(prompter.labels, []string{"alpha"}) {
		t.Fatalf("template labels = %v, want [alpha]", prompter.labels)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".codex", "agents", "reviewer.toml")); err != nil {
		t.Fatalf("expected codex agent file: %v", err)
	}
}

func TestUseRemoteRepositorySortsAndUsesSelectedTemplate(t *testing.T) {
	workDir := t.TempDir()
	repoDir := t.TempDir()
	writeRemoteTemplate(t, repoDir, "beta", remoteTemplate("beta-flow", "# Beta\n"))
	writeRemoteTemplate(t, repoDir, "alpha", remoteTemplate("alpha-flow", "# Alpha\n"))

	prompter := &remotePrompter{selectedLabel: "beta"}
	application := remoteAppForTest(workDir, &fakeGitCloner{sourceDir: repoDir})
	if err := application.Use(t.Context(), "https://example.test/repo.git", prompter); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(prompter.labels, []string{"alpha", "beta"}) {
		t.Fatalf("template labels = %v, want [alpha beta]", prompter.labels)
	}
	agents, err := os.ReadFile(filepath.Join(workDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "# Beta") {
		t.Fatalf("selected template was not used:\n%s", agents)
	}
}

func TestUseRemoteRepositoryRequiresTemplates(t *testing.T) {
	workDir := t.TempDir()
	repoDir := t.TempDir()

	cloner := &fakeGitCloner{sourceDir: repoDir}
	prompter := &remotePrompter{selectedLabel: "alpha"}
	application := remoteAppForTest(workDir, cloner)
	err := application.Use(t.Context(), "https://example.test/repo.git", prompter)
	if err == nil {
		t.Fatal("expected error")
	}
	assertTempRepoRemoved(t, cloner.dest)
	if !strings.Contains(err.Error(), "no templates found") {
		t.Fatalf("error = %q, want no templates found", err)
	}
	if prompter.templateCalls != 0 {
		t.Fatalf("template prompt calls = %d, want 0", prompter.templateCalls)
	}
}

func remoteAppForTest(workDir string, cloner GitCloner) App {
	return App{
		Registry:  adapter.NewRegistry(codex.Adapter{}),
		Writer:    install.NewWriter(),
		Stdout:    &bytes.Buffer{},
		WorkDir:   workDir,
		HomeDir:   workDir,
		GitCloner: cloner,
	}
}

func assertTempRepoRemoved(t *testing.T, repoDest string) {
	t.Helper()
	if repoDest == "" {
		t.Fatal("git cloner did not receive a destination")
	}
	root := filepath.Dir(repoDest)
	if !strings.HasPrefix(filepath.Base(root), "agentflow-") {
		t.Fatalf("temporary repository root = %q, want prefix agentflow-", root)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary repository root still exists or could not be inspected: %v", err)
	}
}

func writeRemoteTemplate(t *testing.T, repoDir, name, content string) {
	t.Helper()
	path := filepath.Join(repoDir, templateRepoDir, name, "template.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func remoteTemplate(id, instructions string) string {
	return `
id: ` + id + `
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
    ` + strings.ReplaceAll(instructions, "\n", "\n    ") + `
`
}

type remotePrompter struct {
	selectedLabel string
	templateCalls int
	labels        []string
}

func (p *remotePrompter) ChooseTemplate(options []builder.TemplateOption) (string, error) {
	p.templateCalls++
	p.labels = p.labels[:0]
	for _, option := range options {
		p.labels = append(p.labels, option.Label)
		if option.Label == p.selectedLabel {
			return option.Value, nil
		}
	}
	return options[0].Value, nil
}

func (p *remotePrompter) ChooseTarget([]builder.TargetOption) (binding.Target, error) {
	return binding.TargetCodex, nil
}

func (p *remotePrompter) AskModel(string, string) (string, error) {
	return "gpt-test", nil
}

func (p *remotePrompter) ChooseScope() (binding.Scope, error) {
	return binding.ScopeProject, nil
}

func (p *remotePrompter) Confirm(string) (bool, error) {
	return true, nil
}

type fakeGitCloner struct {
	sourceDir string
	dest      string
}

func (c *fakeGitCloner) Clone(_ context.Context, _, dest string) error {
	c.dest = dest
	return copyTree(c.sourceDir, dest)
}

func copyTree(source, dest string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
