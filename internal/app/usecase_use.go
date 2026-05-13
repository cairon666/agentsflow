package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/builder"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/schema"
	flowtemplate "github.com/cairon666/agentsflow/internal/template"
)

// Use loads a template, asks the user for choices, renders, and installs files.
func (a App) Use(ctx context.Context, source string, prompter builder.Prompter) error {
	path, cleanup, err := a.resolveTemplateSource(ctx, source, prompter)
	if err != nil {
		return err
	}
	defer cleanup()

	flow, err := flowtemplate.LoadFile(path)
	if err != nil {
		return err
	}
	diags := schema.Validate(flow)
	if len(diags) > 0 {
		fmt.Fprint(a.Stdout, diagnostic.FormatMany(diags))
	}
	if diagnostic.HasErrors(diags) {
		return fmt.Errorf("template validation failed")
	}
	irFlow := schema.ToIR(flow)
	choices, err := builder.Run(irFlow, targetOptions(a.Registry), prompter, a.Stdout)
	if err != nil {
		return err
	}
	targetAdapter, err := a.Registry.Get(string(choices.Target))
	if err != nil {
		return err
	}
	renderInput := adapter.RenderInput{
		Flow:    irFlow,
		Models:  choices.Models,
		Scope:   choices.Scope,
		WorkDir: a.WorkDir,
		HomeDir: a.HomeDir,
	}
	targetDiags := targetAdapter.Validate(ctx, irFlow)
	if len(targetDiags) > 0 {
		fmt.Fprint(a.Stdout, diagnostic.FormatMany(targetDiags))
	}
	if diagnostic.HasErrors(targetDiags) {
		return fmt.Errorf("target validation failed")
	}
	plan, renderDiags := targetAdapter.Render(ctx, renderInput)
	if len(renderDiags) > 0 {
		fmt.Fprint(a.Stdout, diagnostic.FormatMany(renderDiags))
	}
	if diagnostic.HasErrors(renderDiags) {
		return fmt.Errorf("target rendering failed")
	}
	summary := builder.Summary(plan)
	if plan.HasConflicts() {
		fmt.Fprintln(a.Stdout, summary)
		return fmt.Errorf("install plan has conflicts; no files were written")
	}
	ok, err := prompter.Confirm(summary)
	if err != nil {
		return fmt.Errorf("confirm install: %w", err)
	}
	if !ok {
		fmt.Fprintln(a.Stdout, "Cancelled. No files were written.")
		return nil
	}
	fmt.Fprintln(a.Stdout, summary)
	if err := a.Writer.Apply(plan); err != nil {
		return err
	}
	fmt.Fprintln(a.Stdout, "Done.")
	return nil
}

func targetOptions(registry adapter.Registry) []builder.TargetOption {
	options := make([]builder.TargetOption, 0, len(registry.All()))
	for _, item := range registry.All() {
		options = append(options, builder.TargetOption{Value: item.Target(), Label: string(item.Target())})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Value < options[j].Value
	})
	return options
}
