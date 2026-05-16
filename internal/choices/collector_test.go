package choices

import (
	"testing"

	"github.com/cairon666/agentsflow/internal/binding"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

func TestCollectKeepsPreviousChoicesInOutputLog(t *testing.T) {
	history := NewMockHistoryReporter(t)
	history.On("Historyf", "Template: %s\n", []any{"test"}).Once()
	history.On("Historyf", "Target: %s\n", []any{binding.TargetCodex}).Once()
	history.On("Historyf", "Model for %s: %s\n", []any{"main", "model-main"}).Once()
	history.On("Historyf", "Model for %s: %s\n", []any{"code", "model-code"}).Once()
	history.On("Historyf", "Installation scope: %s\n", []any{binding.ScopeProject}).Once()
	history.On("HistorySpace").Once()

	_, err := Collect(
		flowmodel.Flow{
			ID:      "test",
			Version: 1,
			ModelSlots: map[string]flowmodel.ModelSlot{
				"main": {Description: "Main"},
				"code": {Description: "Code"},
			},
			Agents: map[string]flowmodel.Agent{"reviewer": {}},
		},
		[]TargetOption{{Value: binding.TargetCodex, Label: "codex"}},
		logPrompter{},
		history,
	)
	if err != nil {
		t.Fatal(err)
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
