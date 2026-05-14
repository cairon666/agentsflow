package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cairon666/agentsflow/internal/app"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/builder"
	"github.com/cairon666/agentsflow/internal/ir"
)

func newUseCommand(application app.App) *cobra.Command {
	return newUseCommandWithPrompter(application, builder.HuhPrompter{})
}

func newUseCommandWithPrompter(application app.App, fallback builder.Prompter) *cobra.Command {
	options := useOptions{fallback: fallback}
	cmd := &cobra.Command{
		Use:   "use <template|repo>",
		Short: "Interactively install an agentsflow template",
		Args:  validateUseArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompter, err := options.prompter(application)
			if err != nil {
				return err
			}
			return application.Use(cmd.Context(), args[0], prompter)
		},
	}
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return newUseInputError(cmd, err)
	})
	cmd.Flags().StringVar(&options.target, "target", "", "Target CLI tool to configure")
	cmd.Flags().StringArrayVar(&options.binds, "bind", nil, "Bind a model slot to a model, as slot=model")
	cmd.Flags().StringVar(&options.scope, "scope", "", "Installation scope: project or global")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "Approve writing files without prompting")
	return cmd
}

func validateUseArgs(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 1:
		return nil
	case 0:
		return newUseInputError(cmd, errors.New("missing template or repository argument"))
	default:
		return newUseInputError(cmd, fmt.Errorf("expected 1 template or repository argument, received %d", len(args)))
	}
}

func newUseInputError(cmd *cobra.Command, err error) error {
	return fmt.Errorf("%w\n\n%s", err, strings.TrimRight(cmd.UsageString(), "\n"))
}

type useOptions struct {
	target   string
	binds    []string
	scope    string
	yes      bool
	fallback builder.Prompter
}

func (o useOptions) prompter(application app.App) (builder.Prompter, error) {
	models, err := parseModelBindings(o.binds)
	if err != nil {
		return nil, err
	}
	prompter := flagPrompter{
		models:   models,
		yes:      o.yes,
		fallback: o.fallback,
	}
	if strings.TrimSpace(o.target) != "" {
		target, err := application.Registry.Resolve(o.target)
		if err != nil {
			return nil, err
		}
		prompter.target = target
		prompter.hasTarget = true
	}
	if strings.TrimSpace(o.scope) != "" {
		scope, err := parseScope(o.scope)
		if err != nil {
			return nil, err
		}
		prompter.scope = scope
		prompter.hasScope = true
	}
	return prompter, nil
}

func parseModelBindings(values []string) (binding.Models, error) {
	models := binding.Models{}
	for _, value := range values {
		slot, model, ok := strings.Cut(value, "=")
		slot = strings.TrimSpace(slot)
		model = strings.TrimSpace(model)
		if !ok || slot == "" || model == "" {
			return nil, fmt.Errorf("invalid --bind %q; expected slot=model", value)
		}
		if _, exists := models[slot]; exists {
			return nil, fmt.Errorf("duplicate --bind for model slot %q", slot)
		}
		models[slot] = model
	}
	return models, nil
}

func parseScope(value string) (binding.Scope, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(binding.ScopeProject):
		return binding.ScopeProject, nil
	case string(binding.ScopeGlobal):
		return binding.ScopeGlobal, nil
	default:
		return "", fmt.Errorf("invalid --scope %q; expected project or global", value)
	}
}

type flagPrompter struct {
	target    binding.Target
	hasTarget bool
	scope     binding.Scope
	hasScope  bool
	models    binding.Models
	yes       bool
	fallback  builder.Prompter
}

func (p flagPrompter) ValidateModelSlots(slots map[string]ir.ModelSlot) error {
	for slot := range p.models {
		if _, ok := slots[slot]; !ok {
			return fmt.Errorf("unknown model slot %q", slot)
		}
	}
	return nil
}

func (p flagPrompter) ChooseTarget(targets []builder.TargetOption) (binding.Target, error) {
	if p.hasTarget {
		return p.target, nil
	}
	return p.fallback.ChooseTarget(targets)
}

func (p flagPrompter) ChooseTemplate(templates []builder.TemplateOption) (string, error) {
	chooser, ok := p.fallback.(builder.TemplatePrompter)
	if !ok {
		return "", fmt.Errorf("template selection prompt unavailable")
	}
	return chooser.ChooseTemplate(templates)
}

func (p flagPrompter) AskModel(slot, description string) (string, error) {
	if model := p.models[slot]; model != "" {
		return model, nil
	}
	return p.fallback.AskModel(slot, description)
}

func (p flagPrompter) ChooseScope() (binding.Scope, error) {
	if p.hasScope {
		return p.scope, nil
	}
	return p.fallback.ChooseScope()
}

func (p flagPrompter) Confirm(message string) (bool, error) {
	if p.yes {
		return true, nil
	}
	return p.fallback.Confirm(message)
}
