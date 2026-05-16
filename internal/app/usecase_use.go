package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/install"
)

// UseOptions configures the use template flow.
type UseOptions struct {
	DryRun bool
}

// Use loads a template, asks the user for choices, renders, and installs files.
func (a App) Use(ctx context.Context, source string, choices ChoiceCollector) error {
	return a.UseWithOptions(ctx, source, choices, UseOptions{})
}

// UseWithOptions loads a template, asks the user for choices, renders, and installs files.
func (a App) UseWithOptions(ctx context.Context, source string, choices ChoiceCollector, options UseOptions) error {
	a.Reporter.Banner()

	resolved, err := a.TemplateSource.Resolve(ctx, source, choices, a.Reporter)
	if err != nil {
		return err
	}
	if resolved.Cleanup != nil {
		defer resolved.Cleanup()
	}

	loaded, err := a.FlowLoader.LoadFile(resolved.Path)
	if err != nil {
		return err
	}
	if len(loaded.Diagnostics) > 0 {
		a.Reporter.Message(diagnostic.FormatMany(loaded.Diagnostics))
	}
	if diagnostic.HasErrors(loaded.Diagnostics) {
		return fmt.Errorf("template validation failed")
	}
	collected, err := choices.Collect(ctx, loaded.Flow, targetOptions(a.TargetRegistry))
	if err != nil {
		return err
	}
	targetRenderer, err := a.TargetRegistry.Get(string(collected.Target))
	if err != nil {
		return err
	}
	renderInput := RenderInput{
		Flow:    loaded.Flow,
		Models:  collected.Models,
		Scope:   collected.Scope,
		WorkDir: a.WorkDir,
		HomeDir: a.HomeDir,
	}
	targetDiags := targetRenderer.Validate(ctx, renderInput)
	if len(targetDiags) > 0 {
		a.Reporter.Message(diagnostic.FormatMany(targetDiags))
	}
	if diagnostic.HasErrors(targetDiags) {
		return fmt.Errorf("target validation failed")
	}
	artifacts, renderDiags := targetRenderer.Render(ctx, renderInput)
	if len(renderDiags) > 0 {
		a.Reporter.Message(diagnostic.FormatMany(renderDiags))
	}
	if diagnostic.HasErrors(renderDiags) {
		return fmt.Errorf("target rendering failed")
	}
	plan := a.InstallPlanner.Build(artifacts)
	summary := install.FormatSummary(plan)
	if options.DryRun {
		a.Reporter.HistoryBlock(summary)
		if preview := install.FormatDryRunFilePreview(plan); preview != "" {
			a.Reporter.MessageLine(preview)
		}
		if plan.HasConflicts() {
			return fmt.Errorf("install plan has conflicts; no files were written")
		}
		return nil
	}

	if plan.HasConflicts() {
		a.Reporter.HistoryBlock(summary)
		return fmt.Errorf("install plan has conflicts; no files were written")
	}
	ok, err := choices.Confirm(ctx, summary)
	if err != nil {
		return fmt.Errorf("confirm install: %w", err)
	}
	a.Reporter.HistoryBlock(summary)
	if !ok {
		a.Reporter.MessageLine("Cancelled. No files were written.")
		return nil
	}
	if err := a.InstallWriter.Apply(plan); err != nil {
		return err
	}

	a.Reporter.HistorySpace()
	a.Reporter.Historyf("Done.\n")

	return nil
}

func targetOptions(registry TargetRegistry) []TargetOption {
	renderers := registry.All()
	options := make([]TargetOption, 0, len(renderers))
	for _, item := range renderers {
		target := item.Target()
		options = append(options, TargetOption{Value: target, Label: string(target)})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Value < options[j].Value
	})
	return options
}
