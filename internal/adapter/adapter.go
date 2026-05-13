package adapter

import (
	"context"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/ir"
)

// RenderInput contains a validated flow and user choices.
type RenderInput struct {
	Flow    ir.Flow
	Models  binding.Models
	Scope   binding.Scope
	WorkDir string
	HomeDir string
}

// Adapter renders a target-specific install plan.
type Adapter interface {
	Target() binding.Target
	Aliases() []string
	Validate(context.Context, ir.Flow) []diagnostic.Diagnostic
	Render(context.Context, RenderInput) (install.Plan, []diagnostic.Diagnostic)
}
