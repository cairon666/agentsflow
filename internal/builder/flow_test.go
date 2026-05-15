package builder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/ir"
)

func TestRunKeepsPreviousChoicesInOutputLog(t *testing.T) {
	var out bytes.Buffer
	_, err := Run(
		ir.Flow{
			ID:      "test",
			Version: 1,
			ModelSlots: map[string]ir.ModelSlot{
				"main": {Description: "Main"},
				"code": {Description: "Code"},
			},
			Agents: map[string]ir.Agent{"reviewer": {}},
		},
		[]TargetOption{{Value: binding.TargetCodex, Label: "codex"}},
		logPrompter{},
		&out,
	)
	if err != nil {
		t.Fatal(err)
	}
	output := out.String()
	if strings.Contains(output, Banner()) {
		t.Fatalf("builder output should not include startup banner:\n%s", output)
	}
	for _, want := range []string{
		"Target: codex",
		"Model for main: model-main",
		"Model for code: model-code",
		"Installation scope: project",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestSummaryIncludesConflictFiles(t *testing.T) {
	summary := Summary(install.Plan{
		Target: "claude",
		Scope:  "project",
		Actions: []install.Action{
			{Path: "managed.md", Kind: install.ActionUpdate},
			{Path: ".claude/settings.json", Kind: install.ActionConflict},
		},
	})
	if !strings.Contains(summary, "Conflicts: 1") {
		t.Fatalf("summary missing conflict count:\n%s", summary)
	}
	if !strings.Contains(summary, ".claude/settings.json") {
		t.Fatalf("summary missing conflict path:\n%s", summary)
	}
}

type logPrompter struct{}

func (logPrompter) ChooseTarget([]TargetOption) (binding.Target, error) {
	return binding.TargetCodex, nil
}

func (logPrompter) AskModel(slot, _ string) (string, error) {
	return "model-" + slot, nil
}

func (logPrompter) ChooseScope() (binding.Scope, error) {
	return binding.ScopeProject, nil
}

func (logPrompter) Confirm(string) (bool, error) {
	return true, nil
}
