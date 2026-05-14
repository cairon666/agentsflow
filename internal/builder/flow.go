package builder

import (
	"fmt"
	"io"
	"sort"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/ir"
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

// Choices are collected by the interactive builder.
type Choices struct {
	Target binding.Target
	Scope  binding.Scope
	Models binding.Models
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
	ValidateModelSlots(map[string]ir.ModelSlot) error
}

// Run collects all decisions needed to render a flow.
func Run(flow ir.Flow, targets []TargetOption, prompter Prompter, out io.Writer) (Choices, error) {
	fmt.Fprintf(out, "┌   agentsflow\n")
	fmt.Fprintf(out, "│\n")
	fmt.Fprintf(out, "◇  Template: %s (version %d)\n", flow.ID, flow.Version)
	fmt.Fprintf(out, "◇  Agents: %d\n", len(flow.Agents))
	fmt.Fprintf(out, "◇  Model slots: %d\n", len(flow.ModelSlots))
	if validator, ok := prompter.(ModelSlotValidator); ok {
		if err := validator.ValidateModelSlots(flow.ModelSlots); err != nil {
			return Choices{}, fmt.Errorf("validate model bindings: %w", err)
		}
	}
	target, err := prompter.ChooseTarget(targets)
	if err != nil {
		return Choices{}, fmt.Errorf("choose target: %w", err)
	}
	fmt.Fprintf(out, "◇  Selected target: %s\n", target)
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
		fmt.Fprintf(out, "◇  Slot %s: %s\n", slot, model)
	}
	scope, err := prompter.ChooseScope()
	if err != nil {
		return Choices{}, fmt.Errorf("choose scope: %w", err)
	}
	fmt.Fprintf(out, "◇  Installation scope: %s\n\n", scope)
	return Choices{Target: target, Scope: scope, Models: models}, nil
}

// Banner returns the startup ASCII art.
func Banner() string {
	return ` █████╗  ██████╗ ███████╗███╗   ██╗████████╗███████╗██╗      ██████╗ ██╗    ██╗
██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝██╔════╝██║     ██╔═══██╗██║    ██║
███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║   █████╗  ██║     ██║   ██║██║ █╗ ██║
██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║   ██╔══╝  ██║     ██║   ██║██║███╗██║
██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║   ██║     ███████╗╚██████╔╝╚███╔███╔╝
╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ 

`
}

// Summary renders an install plan summary.
func Summary(plan install.Plan) string {
	counts := map[install.ActionKind]int{}
	for _, action := range plan.Actions {
		counts[action.Kind]++
	}
	summary := fmt.Sprintf(
		"Target: %s\nScope: %s\nCreate: %d\nUpdate: %d\nSkip: %d\nConflicts: %d\n",
		plan.Target,
		plan.Scope,
		counts[install.ActionCreate],
		counts[install.ActionUpdate],
		counts[install.ActionSkip],
		counts[install.ActionConflict],
	)
	if counts[install.ActionConflict] > 0 {
		summary += "\nConflict files:\n"
		for _, action := range plan.Actions {
			if action.Kind == install.ActionConflict {
				summary += fmt.Sprintf("- %s\n", action.Path)
			}
		}
	}
	return summary
}

func sortedSlots(flow ir.Flow) []string {
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
