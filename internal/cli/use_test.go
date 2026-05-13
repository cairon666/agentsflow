package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/adapter/codex"
	"github.com/cairon666/agentsflow/internal/app"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/builder"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/ir"
)

func TestUseCommandAcceptsFlags(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, singleSlotTemplate)
	var stdout bytes.Buffer
	application := appForUseTest(workDir, &stdout, codex.Adapter{})

	cmd := newUseCommandWithPrompter(application, failingPrompter{})
	cmd.SetArgs([]string{
		templatePath,
		"--target", "codex",
		"--bind", "main=sonnet",
		"--scope", "project",
		"--yes",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	config, err := os.ReadFile(filepath.Join(workDir, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "model = 'sonnet'") {
		t.Fatalf("config did not use bound model:\n%s", config)
	}
	if !strings.Contains(stdout.String(), "Done.") {
		t.Fatalf("stdout missing completion:\n%s", stdout.String())
	}
}

func TestUseCommandPromptsForMissingFlags(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, twoSlotTemplate)
	var stdout bytes.Buffer
	fallback := &recordingPrompter{models: map[string]string{"code": "opus"}}

	cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout, codex.Adapter{}), fallback)
	cmd.SetArgs([]string{
		templatePath,
		"--target", "codex",
		"--bind", "main=sonnet",
		"--yes",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fallback.targetCalls != 0 {
		t.Fatalf("fallback target prompt was called %d times", fallback.targetCalls)
	}
	if fallback.scopeCalls != 1 {
		t.Fatalf("fallback scope prompt calls = %d, want 1", fallback.scopeCalls)
	}
	if fallback.confirmCalls != 0 {
		t.Fatalf("fallback confirm prompt was called %d times", fallback.confirmCalls)
	}
	output := stdout.String()
	for _, want := range []string{
		"Slot main: sonnet",
		"Slot code: opus",
		"Installation scope: project",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout missing %q:\n%s", want, output)
		}
	}
}

func TestUseCommandRejectsInvalidFlags(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, singleSlotTemplate)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid bind",
			args: []string{templatePath, "--target", "codex", "--bind", "main", "--scope", "project", "--yes"},
			want: "invalid --bind",
		},
		{
			name: "duplicate bind",
			args: []string{templatePath, "--target", "codex", "--bind", "main=sonnet", "--bind", "main=opus", "--scope", "project", "--yes"},
			want: "duplicate --bind",
		},
		{
			name: "unknown bind slot",
			args: []string{templatePath, "--target", "codex", "--bind", "missing=sonnet", "--scope", "project", "--yes"},
			want: "unknown model slot",
		},
		{
			name: "invalid scope",
			args: []string{templatePath, "--target", "codex", "--bind", "main=sonnet", "--scope", "workspace", "--yes"},
			want: "invalid --scope",
		},
		{
			name: "invalid target",
			args: []string{templatePath, "--target", "missing", "--bind", "main=sonnet", "--scope", "project", "--yes"},
			want: "unknown target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout, codex.Adapter{}), failingPrompter{})
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want contains %q", err, tt.want)
			}
		})
	}
}

func TestUseCommandYesDoesNotBypassConflicts(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, singleSlotTemplate)
	var stdout bytes.Buffer
	cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout, conflictAdapter{}), failingPrompter{})
	cmd.SetArgs([]string{
		templatePath,
		"--target", "conflict",
		"--bind", "main=sonnet",
		"--scope", "project",
		"--yes",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "install plan has conflicts") {
		t.Fatalf("error = %q", err)
	}
	if strings.Contains(stdout.String(), "Done.") {
		t.Fatalf("conflicting install completed unexpectedly:\n%s", stdout.String())
	}
}

func TestUseCommandHelpShowsFlags(t *testing.T) {
	var out bytes.Buffer
	cmd := newUseCommandWithPrompter(appForUseTest(t.TempDir(), &bytes.Buffer{}, codex.Adapter{}), failingPrompter{})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--target", "--bind", "--scope", "--yes"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, out.String())
		}
	}
}

func TestUseCommandYesDoesNotBypassTemplateSelection(t *testing.T) {
	fallback := &templateRecordingPrompter{}
	options := useOptions{yes: true, fallback: fallback}
	prompter, err := options.prompter(appForUseTest(t.TempDir(), &bytes.Buffer{}, codex.Adapter{}))
	if err != nil {
		t.Fatal(err)
	}
	chooser, ok := prompter.(builder.TemplatePrompter)
	if !ok {
		t.Fatal("prompter does not support template selection")
	}
	selected, err := chooser.ChooseTemplate([]builder.TemplateOption{
		{Value: "/tmp/template.yaml", Label: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected != "/tmp/template.yaml" {
		t.Fatalf("selected template = %q", selected)
	}
	if fallback.templateCalls != 1 {
		t.Fatalf("template prompt calls = %d, want 1", fallback.templateCalls)
	}
}

func appForUseTest(workDir string, stdout *bytes.Buffer, adapters ...adapter.Adapter) app.App {
	return app.App{
		Registry: adapter.NewRegistry(adapters...),
		Writer:   install.NewWriter(),
		Stdout:   stdout,
		WorkDir:  workDir,
		HomeDir:  workDir,
	}
}

func writeUseTemplate(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

type failingPrompter struct{}

func (failingPrompter) ChooseTarget([]builder.TargetOption) (binding.Target, error) {
	return "", errors.New("unexpected target prompt")
}

func (failingPrompter) AskModel(string, string) (string, error) {
	return "", errors.New("unexpected model prompt")
}

func (failingPrompter) ChooseScope() (binding.Scope, error) {
	return "", errors.New("unexpected scope prompt")
}

func (failingPrompter) Confirm(string) (bool, error) {
	return false, errors.New("unexpected confirm prompt")
}

type recordingPrompter struct {
	models       map[string]string
	targetCalls  int
	scopeCalls   int
	confirmCalls int
}

func (p *recordingPrompter) ChooseTarget([]builder.TargetOption) (binding.Target, error) {
	p.targetCalls++
	return binding.TargetCodex, nil
}

func (p *recordingPrompter) AskModel(slot, _ string) (string, error) {
	if model := p.models[slot]; model != "" {
		return model, nil
	}
	return "", errors.New("unexpected model prompt")
}

func (p *recordingPrompter) ChooseScope() (binding.Scope, error) {
	p.scopeCalls++
	return binding.ScopeProject, nil
}

func (p *recordingPrompter) Confirm(string) (bool, error) {
	p.confirmCalls++
	return true, nil
}

type templateRecordingPrompter struct {
	recordingPrompter
	templateCalls int
}

func (p *templateRecordingPrompter) ChooseTemplate(options []builder.TemplateOption) (string, error) {
	p.templateCalls++
	return options[0].Value, nil
}

type conflictAdapter struct{}

func (conflictAdapter) Target() binding.Target { return binding.Target("conflict") }

func (conflictAdapter) Aliases() []string { return nil }

func (conflictAdapter) Validate(context.Context, ir.Flow) []diagnostic.Diagnostic { return nil }

func (conflictAdapter) Render(context.Context, adapter.RenderInput) (install.Plan, []diagnostic.Diagnostic) {
	return install.Plan{
		Target: "conflict",
		Scope:  "project",
		Actions: []install.Action{
			{Path: "manual.md", Kind: install.ActionConflict},
		},
	}, nil
}

const singleSlotTemplate = `
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

const twoSlotTemplate = `
id: test-flow
version: 1
model_slots:
  main:
    description: Main model
  code:
    description: Code model
permission_profiles:
  read:
    description: Read profile
    capabilities:
      read_files: allow
      edit_files: deny
agents:
  reviewer:
    description: Reviews code
    model_slot: code
    reasoning_effort: medium
    permission_profile: read
    prompt: Review code.
instructions:
  AGENTS.md: |
    # Test
`
