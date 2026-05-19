package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cairon666/agentsflow/internal/app"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/choices"
	"github.com/cairon666/agentsflow/internal/ui/terminal"
)

func newExportCommand(application app.App) *cobra.Command {
	return newExportCommandWithPrompter(application, terminal.HuhPrompter{})
}

func newExportCommandWithPrompter(application app.App, fallback choices.ExportPrompter) *cobra.Command {
	options := exportOptions{fallback: fallback}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export native agent CLI config to an agentsflow template",
		Args:  validateExportArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			prompter, err := options.prompter(application)
			if err != nil {
				return newExportInputError(cmd, err)
			}
			return application.ExportWithOptions(cmd.Context(), exportChoiceCollector{
				prompter: prompter,
				reporter: application.Reporter,
			}, app.ExportOptions{})
		},
	}
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return newExportInputError(cmd, err)
	})
	cmd.Flags().StringVar(&options.source, "source", "", "Source CLI config to export: codex, claude, or opencode")
	cmd.Flags().StringVar(&options.scope, "scope", "", "Export scope: project or global")
	cmd.Flags().StringVar(&options.output, "output", "", "Output template path")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "Approve writing the output file without prompting")
	return cmd
}

func validateExportArgs(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 0:
		return nil
	default:
		return newExportInputError(cmd, fmt.Errorf("expected no arguments, received %d", len(args)))
	}
}

func newExportInputError(cmd *cobra.Command, err error) error {
	return fmt.Errorf("%w\n\n%s", err, strings.TrimRight(cmd.UsageString(), "\n"))
}

type exportOptions struct {
	source   string
	scope    string
	output   string
	yes      bool
	fallback choices.ExportPrompter
}

func (o exportOptions) prompter(application app.App) (choices.ExportPrompter, error) {
	prompter := exportFlagPrompter{
		yes:      o.yes,
		fallback: o.fallback,
	}
	if strings.TrimSpace(o.source) != "" {
		source, err := application.ExporterRegistry.Resolve(o.source)
		if err != nil {
			return nil, err
		}
		prompter.source = source
		prompter.hasSource = true
	}
	if strings.TrimSpace(o.scope) != "" {
		scope, err := parseScope(o.scope)
		if err != nil {
			return nil, err
		}
		prompter.scope = scope
		prompter.hasScope = true
	}
	if strings.TrimSpace(o.output) != "" {
		prompter.output = o.output
		prompter.hasOutput = true
	}
	return prompter, nil
}

type exportFlagPrompter struct {
	source    binding.Target
	hasSource bool
	scope     binding.Scope
	hasScope  bool
	output    string
	hasOutput bool
	yes       bool
	fallback  choices.ExportPrompter
}

func (p exportFlagPrompter) ChooseSource(sources []choices.SourceOption) (binding.Target, error) {
	if p.hasSource {
		return p.source, nil
	}
	if p.fallback == nil {
		return "", errors.New("source selection prompt unavailable")
	}
	return p.fallback.ChooseSource(sources)
}

func (p exportFlagPrompter) ChooseScope() (binding.Scope, error) {
	if p.hasScope {
		return p.scope, nil
	}
	if p.fallback == nil {
		return "", errors.New("scope selection prompt unavailable")
	}
	return p.fallback.ChooseScope()
}

func (p exportFlagPrompter) AskOutputPath(defaultPath string) (string, error) {
	if p.hasOutput {
		return p.output, nil
	}
	if p.fallback == nil {
		return "", errors.New("output path prompt unavailable")
	}
	return p.fallback.AskOutputPath(defaultPath)
}

func (p exportFlagPrompter) Confirm(message string) (bool, error) {
	if p.yes {
		return true, nil
	}
	if p.fallback == nil {
		return false, errors.New("confirmation prompt unavailable")
	}
	return p.fallback.Confirm(message)
}
