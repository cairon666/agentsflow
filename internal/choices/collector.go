package choices

import (
	"fmt"
	"sort"

	"github.com/cairon666/agentsflow/internal/binding"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

// TargetOption is shown to the user in the target selection step.
type TargetOption struct {
	Value binding.Target
	Label string
}

// TemplateOption is shown to the user when a repository contains templates.
type TemplateOption struct {
	Value string
	Label string
}

// Choices are collected before rendering a flow.
type Choices struct {
	Target binding.Target
	Scope  binding.Scope
	Models binding.Models
}

// HistoryReporter records stable user choices outside transient prompts.
type HistoryReporter interface {
	Historyf(format string, args ...any)
	HistorySpace()
}

// Prompter collects decisions from the user.
type Prompter interface {
	ChooseTarget([]TargetOption) (binding.Target, error)
	AskModel(slot, description string) (string, error)
	ChooseScope() (binding.Scope, error)
	Confirm(message string) (bool, error)
}

// TemplatePrompter chooses a template from a repository source.
type TemplatePrompter interface {
	ChooseTemplate([]TemplateOption) (string, error)
}

// ModelSlotValidator validates preconfigured model bindings against a flow.
type ModelSlotValidator interface {
	ValidateModelSlots(map[string]flowmodel.ModelSlot) error
}

// Collect collects all decisions needed to render a flow.
func Collect(flow flowmodel.Flow, targets []TargetOption, prompter Prompter, history HistoryReporter) (Choices, error) {
	history.Historyf("Template: %s\n", flow.ID)

	if validator, ok := prompter.(ModelSlotValidator); ok {
		if err := validator.ValidateModelSlots(flow.ModelSlots); err != nil {
			return Choices{}, fmt.Errorf("validate model bindings: %w", err)
		}
	}
	target, err := prompter.ChooseTarget(targets)
	if err != nil {
		return Choices{}, fmt.Errorf("choose target: %w", err)
	}
	history.Historyf("Target: %s\n", target)

	models := make(binding.Models, len(flow.ModelSlots))
	for _, slot := range sortedSlots(flow) {
		model, err := prompter.AskModel(slot, flow.ModelSlots[slot].Description)
		if err != nil {
			return Choices{}, fmt.Errorf("model for slot %q: %w", slot, err)
		}
		if model == "" {
			return Choices{}, fmt.Errorf("model for slot %q is required", slot)
		}
		models[slot] = model
		history.Historyf("Model for %s: %s\n", slot, model)
	}
	scope, err := prompter.ChooseScope()
	if err != nil {
		return Choices{}, fmt.Errorf("choose scope: %w", err)
	}
	history.Historyf("Installation scope: %s\n", scope)
	history.HistorySpace()

	return Choices{Target: target, Scope: scope, Models: models}, nil
}

func sortedSlots(flow flowmodel.Flow) []string {
	preferred := []string{"main", "scout", "reasoning", "research", "execution", "code"}
	slots := make([]string, 0, len(flow.ModelSlots))
	seen := make(map[string]struct{}, len(flow.ModelSlots))
	for _, slot := range preferred {
		if _, ok := flow.ModelSlots[slot]; ok {
			slots = append(slots, slot)
			seen[slot] = struct{}{}
		}
	}
	rest := make([]string, 0, len(flow.ModelSlots))
	for slot := range flow.ModelSlots {
		if _, ok := seen[slot]; ok {
			continue
		}
		rest = append(rest, slot)
	}
	sort.Strings(rest)
	slots = append(slots, rest...)
	return slots
}
