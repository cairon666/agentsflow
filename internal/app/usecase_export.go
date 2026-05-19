package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
)

// ExportOptions configures the native config export flow.
type ExportOptions struct{}

// Export reads a native agent CLI config and writes an agentsflow template.
func (a App) Export(ctx context.Context, choices ExportChoiceCollector) error {
	return a.ExportWithOptions(ctx, choices, ExportOptions{})
}

// ExportWithOptions reads a native agent CLI config and writes an agentsflow template.
func (a App) ExportWithOptions(ctx context.Context, choices ExportChoiceCollector, _ ExportOptions) error {
	a.Reporter.Banner()

	collected, err := choices.CollectExport(ctx, exportSourceOptions(a.ExporterRegistry))
	if err != nil {
		return err
	}
	sourceExporter, err := a.ExporterRegistry.Get(string(collected.Source))
	if err != nil {
		return err
	}
	exported, err := sourceExporter.Export(ctx, ExportInput{
		Source:  collected.Source,
		Scope:   collected.Scope,
		WorkDir: a.WorkDir,
		HomeDir: a.HomeDir,
	})
	if len(exported.Diagnostics) > 0 {
		a.Reporter.HistoryBlock(diagnostic.FormatMany(exported.Diagnostics))
	}
	if err != nil {
		return err
	}
	if diagnostic.HasErrors(exported.Diagnostics) {
		return fmt.Errorf("source export failed")
	}

	validationDiags := flowmodel.ValidateSpec(exported.Spec)
	if len(validationDiags) > 0 {
		a.Reporter.HistoryBlock(diagnostic.FormatMany(validationDiags))
	}
	if diagnostic.HasErrors(validationDiags) {
		return fmt.Errorf("exported template validation failed")
	}
	content, err := a.SpecEncoder.EncodeSpec(exported.Spec)
	if err != nil {
		return err
	}

	outputPath := collected.Output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(a.WorkDir, outputPath)
	}
	artifacts := install.ArtifactSet{
		Target: "export",
		Scope:  string(collected.Scope),
		Files: []install.DesiredFile{
			{Path: outputPath, Content: content, Strategy: install.StrategyOverwrite},
		},
	}
	plan := a.InstallPlanner.Build(artifacts)
	summary := install.FormatSummary(plan)
	if plan.HasConflicts() {
		a.Reporter.HistoryBlock(summary)
		return fmt.Errorf("export plan has conflicts; no files were written")
	}
	ok, err := choices.Confirm(ctx, summary)
	if err != nil {
		return fmt.Errorf("confirm export: %w", err)
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

func exportSourceOptions(registry ExporterRegistry) []ExportSourceOption {
	exporters := registry.All()
	options := make([]ExportSourceOption, 0, len(exporters))
	for _, item := range exporters {
		source := item.Source()
		options = append(options, ExportSourceOption{Value: source, Label: string(source)})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Value < options[j].Value
	})
	return options
}
