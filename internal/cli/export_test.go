package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/choices"
)

func TestExportCommandAcceptsFlags(t *testing.T) {
	workDir := t.TempDir()
	writeNativeFile(t, filepath.Join(workDir, "AGENTS.md"), "# Shared\n")
	writeNativeFile(t, filepath.Join(workDir, ".codex", "agents", "reviewer.toml"), `
name = 'reviewer'
description = 'Reviews code.'
developer_instructions = 'Review.'
`)
	var stdout bytes.Buffer
	outputPath := filepath.Join(workDir, "exported.yaml")

	cmd := newExportCommandWithPrompter(appForUseTest(workDir, &stdout), failingExportPrompter{})
	cmd.SetArgs([]string{
		"--source", "codex",
		"--scope", "project",
		"--output", outputPath,
		"--yes",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"id: exported-codex-project",
		"main:",
		"reviewer-model:",
		"reviewer-permissions:",
		"Done.",
	} {
		if !strings.Contains(string(content)+stdout.String(), want) {
			t.Fatalf("export output missing %q\nfile:\n%s\nstdout:\n%s", want, content, stdout.String())
		}
	}
}

func TestExportCommandPromptsForMissingFlags(t *testing.T) {
	workDir := t.TempDir()
	writeNativeFile(t, filepath.Join(workDir, ".codex", "agents", "reviewer.toml"), `
name = 'reviewer'
description = 'Reviews code.'
developer_instructions = 'Review.'
`)
	var stdout bytes.Buffer
	fallback := &recordingExportPrompter{}

	cmd := newExportCommandWithPrompter(appForUseTest(workDir, &stdout), fallback)
	cmd.SetArgs([]string{"--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fallback.sourceCalls != 1 {
		t.Fatalf("source prompt calls = %d, want 1", fallback.sourceCalls)
	}
	if fallback.scopeCalls != 1 {
		t.Fatalf("scope prompt calls = %d, want 1", fallback.scopeCalls)
	}
	if fallback.outputCalls != 1 {
		t.Fatalf("output prompt calls = %d, want 1", fallback.outputCalls)
	}
	if fallback.confirmCalls != 0 {
		t.Fatalf("confirm prompt calls = %d, want 0", fallback.confirmCalls)
	}
	if _, err := os.Stat(filepath.Join(workDir, choices.DefaultExportOutput)); err != nil {
		t.Fatalf("expected default output file: %v", err)
	}
	assertOutputContains(t, stdout.String(), []string{
		"Source: codex",
		"Export scope: project",
		"Output: agentsflow.yaml",
		"Done.",
	})
	cleanOutput := stripANSISequences(stdout.String())
	if !strings.Contains(cleanOutput, "┃  [warning] Codex shared instructions file is missing") {
		t.Fatalf("warning was not history-prefixed:\n%s", cleanOutput)
	}
}

func TestExportCommandRejectsInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid source",
			args: []string{"--source", "missing", "--scope", "project", "--output", "agentsflow.yaml", "--yes"},
			want: "unknown source",
		},
		{
			name: "invalid scope",
			args: []string{"--source", "codex", "--scope", "workspace", "--output", "agentsflow.yaml", "--yes"},
			want: "invalid --scope",
		},
		{
			name: "extra argument",
			args: []string{"extra"},
			want: "expected no arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			cmd := newExportCommandWithPrompter(appForUseTest(t.TempDir(), &stdout), failingExportPrompter{})
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			got := err.Error()
			for _, want := range []string{tt.want, "Usage:", "export [flags]"} {
				if !strings.Contains(got, want) {
					t.Fatalf("error missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestExportCommandHelpShowsFlags(t *testing.T) {
	var out bytes.Buffer
	cmd := newExportCommandWithPrompter(appForUseTest(t.TempDir(), &bytes.Buffer{}), failingExportPrompter{})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--source", "--scope", "--output", "--yes"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, out.String())
		}
	}
}

type failingExportPrompter struct{}

func (failingExportPrompter) ChooseSource([]choices.SourceOption) (binding.Target, error) {
	return "", errors.New("unexpected source prompt")
}

func (failingExportPrompter) ChooseScope() (binding.Scope, error) {
	return "", errors.New("unexpected scope prompt")
}

func (failingExportPrompter) AskOutputPath(string) (string, error) {
	return "", errors.New("unexpected output prompt")
}

func (failingExportPrompter) Confirm(string) (bool, error) {
	return false, errors.New("unexpected confirm prompt")
}

type recordingExportPrompter struct {
	sourceCalls  int
	scopeCalls   int
	outputCalls  int
	confirmCalls int
}

func (p *recordingExportPrompter) ChooseSource([]choices.SourceOption) (binding.Target, error) {
	p.sourceCalls++
	return binding.TargetCodex, nil
}

func (p *recordingExportPrompter) ChooseScope() (binding.Scope, error) {
	p.scopeCalls++
	return binding.ScopeProject, nil
}

func (p *recordingExportPrompter) AskOutputPath(string) (string, error) {
	p.outputCalls++
	return "", nil
}

func (p *recordingExportPrompter) Confirm(string) (bool, error) {
	p.confirmCalls++
	return true, nil
}

func writeNativeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSISequences(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}
