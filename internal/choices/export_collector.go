package choices

import (
	"fmt"
	"strings"

	"github.com/cairon666/agentsflow/internal/binding"
)

const DefaultExportOutput = "agentsflow.yaml"

// SourceOption is shown to the user in the export source selection step.
type SourceOption struct {
	Value binding.Target
	Label string
}

// ExportChoices are collected before exporting native config.
type ExportChoices struct {
	Source binding.Target
	Scope  binding.Scope
	Output string
}

// ExportPrompter collects decisions for native config export.
type ExportPrompter interface {
	ChooseSource([]SourceOption) (binding.Target, error)
	ChooseScope() (binding.Scope, error)
	AskOutputPath(defaultPath string) (string, error)
	Confirm(message string) (bool, error)
}

// CollectExport collects all decisions needed to export native config.
func CollectExport(sources []SourceOption, prompter ExportPrompter, history HistoryReporter) (ExportChoices, error) {
	source, err := prompter.ChooseSource(sources)
	if err != nil {
		return ExportChoices{}, fmt.Errorf("choose source: %w", err)
	}
	history.Historyf("Source: %s\n", source)

	scope, err := prompter.ChooseScope()
	if err != nil {
		return ExportChoices{}, fmt.Errorf("choose scope: %w", err)
	}
	history.Historyf("Export scope: %s\n", scope)

	output, err := prompter.AskOutputPath(DefaultExportOutput)
	if err != nil {
		return ExportChoices{}, fmt.Errorf("output path: %w", err)
	}
	output = strings.TrimSpace(output)
	if output == "" {
		output = DefaultExportOutput
	}
	history.Historyf("Output: %s\n", output)
	history.HistorySpace()

	return ExportChoices{Source: source, Scope: scope, Output: output}, nil
}
