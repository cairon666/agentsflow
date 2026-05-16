package binding

import flowmodel "github.com/cairon666/agentsflow/internal/flow"

// ResolvedFlow combines a flow with user choices collected by the choice collector.
type ResolvedFlow struct {
	Flow    flowmodel.Flow
	Target  Target
	Scope   Scope
	Models  Models
	WorkDir string
	HomeDir string
}
