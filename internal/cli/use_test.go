package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/app"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/choices"
	"github.com/cairon666/agentsflow/internal/composition"
)

func TestUseCommandAcceptsFlags(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, implicitMainTemplate)
	var stdout bytes.Buffer
	application := appForUseTest(workDir, &stdout)

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
	assertOutputContains(t, stdout.String(), []string{
		"Template: test-flow",
		"Target: codex",
		"Model for main: sonnet",
		"Installation scope: project",
		"Done.",
	})
}

func TestUseCommandPromptsForMissingFlags(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, twoSlotTemplate)
	var stdout bytes.Buffer
	fallback := &recordingPrompter{models: map[string]string{"code": "opus"}}

	cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout), fallback)
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
	assertOutputContains(t, output, []string{
		"Template: test-flow",
		"Target: codex",
		"Model for main: sonnet",
		"Model for code: opus",
		"Installation scope: project",
		"Done.",
	})
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
			cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout), failingPrompter{})
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

func TestUseCommandInputErrorsIncludeUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing argument",
			args: []string{"use"},
			want: "missing template or repository argument",
		},
		{
			name: "unknown flag",
			args: []string{"use", "https://github.com/cairon666/agentsflow", "--dont-know-flag"},
			want: "unknown flag: --dont-know-flag",
		},
		{
			name: "extra argument",
			args: []string{"use", "template.yaml", "extra.yaml"},
			want: "expected 1 template or repository argument, received 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand()
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			got := err.Error()
			for _, want := range []string{
				tt.want,
				"Usage:",
				"agentsflow use <template|repo> [flags]",
				"--target",
				"--bind",
				"--scope",
			} {
				if !strings.Contains(got, want) {
					t.Fatalf("error missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestUseCommandYesOverwritesManagedFiles(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, singleSlotTemplate)
	managedPath := filepath.Join(workDir, "AGENTS.md")
	if err := os.WriteFile(managedPath, []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout), failingPrompter{})
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
	content, err := os.ReadFile(managedPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "# Test" {
		t.Fatalf("managed file was not overwritten:\n%s", content)
	}
	if !strings.Contains(stdout.String(), "Done.") {
		t.Fatalf("install did not complete:\n%s", stdout.String())
	}
}

func TestUseCommandHelpShowsFlags(t *testing.T) {
	var out bytes.Buffer
	cmd := newUseCommandWithPrompter(appForUseTest(t.TempDir(), &bytes.Buffer{}), failingPrompter{})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--target", "--bind", "--scope", "--yes", "--dry-run"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, out.String())
		}
	}
}

func TestUseCommandDryRunPrintsPreviewWithoutWritingFiles(t *testing.T) {
	workDir := t.TempDir()
	templatePath := writeUseTemplate(t, workDir, implicitMainTemplate)
	configPath := filepath.Join(workDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	originalConfig := "model = 'old'\ncustom = true\n"
	if err := os.WriteFile(configPath, []byte(originalConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newUseCommandWithPrompter(appForUseTest(workDir, &stdout), failingPrompter{})
	cmd.SetArgs([]string{
		templatePath,
		"--target", "codex",
		"--bind", "main=sonnet",
		"--scope", "project",
		"--dry-run",
		"--yes",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != originalConfig {
		t.Fatalf("dry run changed config:\n%s", content)
	}
	if _, err := os.Stat(filepath.Join(workDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("dry run created AGENTS.md, stat err = %v", err)
	}
	output := stdout.String()
	assertOutputContains(t, output, []string{
		"Files:",
		"AGENTS.md (create)",
		"--- planned content ---",
		"# Test",
		"--- merge diff ---",
		"-model = 'old'",
		"+model = 'sonnet'",
	})
	for _, noise := range []string{
		"Dry run install plan:",
		"Create files:",
		"Clean directories:",
		"Dry run complete. No files were written.",
		"Done.",
	} {
		if strings.Contains(output, noise) {
			t.Fatalf("dry run output contains %q:\n%s", noise, output)
		}
	}
}

func TestUseCommandYesDoesNotBypassTemplateSelection(t *testing.T) {
	fallback := &templateRecordingPrompter{}
	options := useOptions{yes: true, fallback: fallback}
	prompter, err := options.prompter(appForUseTest(t.TempDir(), &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	chooser, ok := prompter.(choices.TemplatePrompter)
	if !ok {
		t.Fatal("prompter does not support template selection")
	}
	selected, err := chooser.ChooseTemplate([]choices.TemplateOption{
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

func appForUseTest(workDir string, stdout *bytes.Buffer) app.App {
	application := composition.NewApp(composition.Config{Stdout: stdout})
	application.WorkDir = workDir
	application.HomeDir = workDir
	return application
}

func assertOutputContains(t *testing.T, output string, values []string) {
	t.Helper()
	for _, want := range values {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout missing %q:\n%s", want, output)
		}
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

func (failingPrompter) ChooseTarget([]choices.TargetOption) (binding.Target, error) {
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

func (p *recordingPrompter) ChooseTarget([]choices.TargetOption) (binding.Target, error) {
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

func (p *templateRecordingPrompter) ChooseTemplate(options []choices.TemplateOption) (string, error) {
	p.templateCalls++
	return options[0].Value, nil
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

const implicitMainTemplate = `
id: test-flow
version: 1
permission_profiles:
  read:
    description: Read profile
    capabilities:
      read_files: allow
      edit_files: deny
agents:
  reviewer:
    description: Reviews code
    reasoning_effort: medium
    permission_profile: read
    prompt: Review code.
instructions:
  AGENTS.md: |
    # Test
`
