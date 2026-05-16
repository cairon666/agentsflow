package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/adapter/codex"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/builder"
	"github.com/cairon666/agentsflow/internal/console"
	"github.com/cairon666/agentsflow/internal/install"
	templatesource "github.com/cairon666/agentsflow/internal/source"
)

func TestUseWritesCodexFilesWithFakePrompter(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	templatePath := filepath.Join(workDir, "template.yaml")
	if err := os.WriteFile(templatePath, []byte(testTemplate), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	application := App{
		Registry: adapter.NewRegistry(codex.Adapter{}),
		Writer:   install.NewWriter(),
		Stdout:   &stdout,
		WorkDir:  workDir,
		HomeDir:  homeDir,
	}
	if err := application.Use(t.Context(), templatePath, fakePrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".codex", "agents", "reviewer.toml")); err != nil {
		t.Fatalf("expected codex agent file: %v", err)
	}
	output := stdout.String()
	if !strings.HasPrefix(output, console.Banner()) {
		t.Fatalf("stdout missing banner:\n%s", output)
	}
	assertOutputContains(t, output, []string{
		"Template: test-flow",
		"Target: codex",
		"Model for main: gpt-test",
		"Installation scope: project",
		"Done.",
	})
}

func TestUseResolvesRemoteSourceWithDefaultResolver(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	repoDir := t.TempDir()
	writeRemoteTemplate(t, repoDir, "alpha", strings.Replace(testTemplate, "# Test", "# Alpha", 1))
	writeRemoteTemplate(t, repoDir, "beta", strings.Replace(testTemplate, "# Test", "# Beta", 1))
	var stdout bytes.Buffer
	cloner := &fakeGitCloner{sourceDir: repoDir}
	prompter := &templateChoosingPrompter{selectedLabel: "beta"}
	application := App{
		Registry:       adapter.NewRegistry(codex.Adapter{}),
		Writer:         install.NewWriter(),
		SourceResolver: templatesource.DefaultResolver{Cloner: cloner},
		Stdout:         &stdout,
		WorkDir:        workDir,
		HomeDir:        homeDir,
	}

	if err := application.Use(t.Context(), "https://example.test/repo.git", prompter); err != nil {
		t.Fatal(err)
	}

	assertTempRepoRemoved(t, cloner.dest)
	if !reflect.DeepEqual(prompter.templateLabels, []string{"alpha", "beta"}) {
		t.Fatalf("template labels = %v, want [alpha beta]", prompter.templateLabels)
	}
	agents, err := os.ReadFile(filepath.Join(workDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "# Beta") {
		t.Fatalf("selected template was not used:\n%s", agents)
	}
	assertOutputContains(t, stdout.String(), []string{
		"Source: https://example.test/repo.git",
		"Template: test-flow",
		"Target: codex",
		"Model for main: gpt-test",
		"Installation scope: project",
		"Done.",
	})
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

type templateChoosingPrompter struct {
	fakePrompter
	selectedLabel  string
	templateLabels []string
}

func (p *templateChoosingPrompter) ChooseTemplate(options []builder.TemplateOption) (string, error) {
	p.templateLabels = p.templateLabels[:0]
	for _, option := range options {
		p.templateLabels = append(p.templateLabels, option.Label)
		if option.Label == p.selectedLabel {
			return option.Value, nil
		}
	}
	return options[0].Value, nil
}

type fakeGitCloner struct {
	sourceDir string
	dest      string
}

func (c *fakeGitCloner) Clone(_ context.Context, _, dest string) error {
	c.dest = dest
	return copyTree(c.sourceDir, dest)
}

func writeRemoteTemplate(t *testing.T, repoDir, name, content string) {
	t.Helper()
	path := filepath.Join(repoDir, ".agentsflow", name, "template.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTempRepoRemoved(t *testing.T, repoDest string) {
	t.Helper()
	if repoDest == "" {
		t.Fatal("git cloner did not receive a destination")
	}
	root := filepath.Dir(repoDest)
	if !strings.HasPrefix(filepath.Base(root), "agentsflow-") {
		t.Fatalf("temporary repository root = %q, want prefix agentsflow-", root)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary repository root still exists or could not be inspected: %v", err)
	}
}

func assertOutputContains(t *testing.T, output string, values []string) {
	t.Helper()
	for _, want := range values {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout missing %q:\n%s", want, output)
		}
	}
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
