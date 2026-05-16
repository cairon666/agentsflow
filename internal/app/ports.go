package app

import (
	"context"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
)

// Reporter owns user-facing output for app use cases.
type Reporter interface {
	Banner()
	Historyf(format string, args ...any)
	HistorySpace()
	Message(args ...any)
	MessageLine(args ...any)
	RunLoading(context.Context, string, func(context.Context) error) error
}

// TemplateOption is shown to the user when a source contains templates.
type TemplateOption struct {
	Value string
	Label string
}

// TemplateChooser chooses a template from a source.
type TemplateChooser interface {
	ChooseTemplate([]TemplateOption) (string, error)
}

// ResolvedSource points to a resolved template and its cleanup hook.
type ResolvedSource struct {
	Path    string
	Cleanup func()
}

// TemplateSource resolves template sources to local template files.
type TemplateSource interface {
	Resolve(context.Context, string, TemplateChooser, Reporter) (ResolvedSource, error)
}

// LoadResult contains a normalized flow and validation diagnostics.
type LoadResult struct {
	Flow        flowmodel.Flow
	Diagnostics []diagnostic.Diagnostic
}

// FlowLoader loads and normalizes a flow template.
type FlowLoader interface {
	LoadFile(path string) (LoadResult, error)
}

// TargetOption is shown to the user in the target selection step.
type TargetOption struct {
	Value binding.Target
	Label string
}

// Choices are collected from the user or flags before rendering.
type Choices struct {
	Target binding.Target
	Scope  binding.Scope
	Models binding.Models
}

// ChoiceCollector collects decisions required for rendering and install.
type ChoiceCollector interface {
	TemplateChooser
	Collect(context.Context, flowmodel.Flow, []TargetOption) (Choices, error)
	Confirm(context.Context, string) (bool, error)
}

// RenderInput contains a validated flow and user choices.
type RenderInput struct {
	Flow    flowmodel.Flow
	Models  binding.Models
	Scope   binding.Scope
	WorkDir string
	HomeDir string
}

// TargetRenderer renders target-specific desired files.
type TargetRenderer interface {
	Target() binding.Target
	Validate(context.Context, RenderInput) []diagnostic.Diagnostic
	Render(context.Context, RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic)
}

// TargetRegistry resolves and lists target renderers.
type TargetRegistry interface {
	Resolve(string) (binding.Target, error)
	Get(string) (TargetRenderer, error)
	All() []TargetRenderer
}

// InstallPlanner builds an install plan from rendered artifacts.
type InstallPlanner interface {
	Build(install.ArtifactSet) install.Plan
}

// InstallWriter applies an install plan.
type InstallWriter interface {
	Apply(install.Plan) error
}
