package terminal

import (
	"fmt"

	"charm.land/huh/v2"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/choices"
)

// HuhPrompter collects answers with huh forms.
type HuhPrompter struct{}

// ChooseTarget asks which CLI target should be generated.
func (p HuhPrompter) ChooseTarget(targets []choices.TargetOption) (binding.Target, error) {
	options := make([]huh.Option[binding.Target], 0, len(targets))
	for _, target := range targets {
		options = append(options, huh.NewOption(target.Label, target.Value))
	}
	var value binding.Target
	if err := huh.NewSelect[binding.Target]().
		Title("Which CLI tool do you want to configure?").
		Options(options...).
		Value(&value).
		Run(); err != nil {
		return "", err
	}
	return value, nil
}

// ChooseSource asks which CLI source should be exported.
func (p HuhPrompter) ChooseSource(sources []choices.SourceOption) (binding.Target, error) {
	options := make([]huh.Option[binding.Target], 0, len(sources))
	for _, source := range sources {
		options = append(options, huh.NewOption(source.Label, source.Value))
	}
	var value binding.Target
	if err := huh.NewSelect[binding.Target]().
		Title("Which CLI config do you want to export?").
		Options(options...).
		Value(&value).
		Run(); err != nil {
		return "", err
	}
	return value, nil
}

// ChooseTemplate asks which repository template should be used.
func (p HuhPrompter) ChooseTemplate(templates []choices.TemplateOption) (string, error) {
	options := make([]huh.Option[string], 0, len(templates))
	for _, template := range templates {
		options = append(options, huh.NewOption(template.Label, template.Value))
	}
	var value string
	if err := huh.NewSelect[string]().
		Title("Which template do you want to use?").
		Options(options...).
		Value(&value).
		Run(); err != nil {
		return "", err
	}
	return value, nil
}

// AskModel asks for a model binding.
func (p HuhPrompter) AskModel(slot, description string) (string, error) {
	var value string
	title := fmt.Sprintf("Model for %s", slot)
	if description != "" {
		title += "\n" + description
	}
	if err := huh.NewInput().
		Title(title).
		Value(&value).
		Run(); err != nil {
		return "", err
	}
	return value, nil
}

// AskOutputPath asks where the exported template should be written.
func (p HuhPrompter) AskOutputPath(defaultPath string) (string, error) {
	var value string
	if err := huh.NewInput().
		Title("Output template path").
		Placeholder(defaultPath).
		Value(&value).
		Run(); err != nil {
		return "", err
	}
	return value, nil
}

// ChooseScope asks where files should be installed.
func (p HuhPrompter) ChooseScope() (binding.Scope, error) {
	var value binding.Scope
	if err := huh.NewSelect[binding.Scope]().
		Title("Installation scope").
		Options(
			huh.NewOption("Project", binding.ScopeProject),
			huh.NewOption("Global", binding.ScopeGlobal),
		).
		Value(&value).
		Run(); err != nil {
		return "", err
	}
	return value, nil
}

// Confirm asks for final confirmation.
func (p HuhPrompter) Confirm(message string) (bool, error) {
	ok := true
	if err := huh.NewConfirm().
		Title(message).
		Affirmative("Write files").
		Negative("Cancel").
		Value(&ok).
		Run(); err != nil {
		return false, err
	}
	return ok, nil
}
