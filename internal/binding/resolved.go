package binding

import "github.com/cairon666/agentsflow/internal/ir"

// ResolvedFlow combines IR with user choices collected by the builder.
type ResolvedFlow struct {
	Flow    ir.Flow
	Target  Target
	Scope   Scope
	Models  Models
	WorkDir string
	HomeDir string
}
