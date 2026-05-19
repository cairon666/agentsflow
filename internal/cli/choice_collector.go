package cli

import (
	"context"
	"fmt"

	"github.com/cairon666/agentsflow/internal/app"
	"github.com/cairon666/agentsflow/internal/choices"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

type choiceCollector struct {
	prompter choices.Prompter
	reporter app.Reporter
}

func (c choiceCollector) ChooseTemplate(options []app.TemplateOption) (string, error) {
	chooser, ok := c.prompter.(choices.TemplatePrompter)
	if !ok {
		return "", fmt.Errorf("template selection prompt unavailable")
	}
	choiceOptions := make([]choices.TemplateOption, 0, len(options))
	for _, option := range options {
		choiceOptions = append(choiceOptions, choices.TemplateOption{
			Value: option.Value,
			Label: option.Label,
		})
	}
	return chooser.ChooseTemplate(choiceOptions)
}

func (c choiceCollector) Collect(_ context.Context, flow flowmodel.Flow, targets []app.TargetOption) (app.Choices, error) {
	choiceTargets := make([]choices.TargetOption, 0, len(targets))
	for _, target := range targets {
		choiceTargets = append(choiceTargets, choices.TargetOption{
			Value: target.Value,
			Label: target.Label,
		})
	}
	collected, err := choices.Collect(flow, choiceTargets, c.prompter, c.reporter)
	if err != nil {
		return app.Choices{}, err
	}
	return app.Choices{Target: collected.Target, Scope: collected.Scope, Models: collected.Models}, nil
}

func (c choiceCollector) Confirm(_ context.Context, message string) (bool, error) {
	return c.prompter.Confirm(message)
}

type exportChoiceCollector struct {
	prompter choices.ExportPrompter
	reporter app.Reporter
}

func (c exportChoiceCollector) CollectExport(_ context.Context, sources []app.ExportSourceOption) (app.ExportChoices, error) {
	choiceSources := make([]choices.SourceOption, 0, len(sources))
	for _, source := range sources {
		choiceSources = append(choiceSources, choices.SourceOption{
			Value: source.Value,
			Label: source.Label,
		})
	}
	collected, err := choices.CollectExport(choiceSources, c.prompter, c.reporter)
	if err != nil {
		return app.ExportChoices{}, err
	}
	return app.ExportChoices{Source: collected.Source, Scope: collected.Scope, Output: collected.Output}, nil
}

func (c exportChoiceCollector) Confirm(_ context.Context, message string) (bool, error) {
	return c.prompter.Confirm(message)
}
