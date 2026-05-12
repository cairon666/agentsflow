package binding

// Scope describes where generated files should be installed.
type Scope string

const (
	// ScopeProject writes files under the current project directory.
	ScopeProject Scope = "project"
	// ScopeGlobal writes files under the target tool's global config directory.
	ScopeGlobal Scope = "global"
)
