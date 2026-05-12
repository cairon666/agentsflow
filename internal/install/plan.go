package install

// ActionKind describes a planned filesystem operation.
type ActionKind string

const (
	// ActionCreate creates a new file.
	ActionCreate ActionKind = "create"
	// ActionUpdate updates a managed file.
	ActionUpdate ActionKind = "update"
	// ActionSkip leaves an identical file untouched.
	ActionSkip ActionKind = "skip"
	// ActionConflict reports an unmanaged existing file.
	ActionConflict ActionKind = "conflict"
)

// Action is one planned filesystem operation.
type Action struct {
	Path      string
	Kind      ActionKind
	Content   []byte
	ManagedBy bool
}

// Plan is the set of files a target adapter wants to install.
type Plan struct {
	Target  string
	Scope   string
	Actions []Action
}

// HasConflicts reports whether any planned action is a conflict.
func (p Plan) HasConflicts() bool {
	for _, action := range p.Actions {
		if action.Kind == ActionConflict {
			return true
		}
	}
	return false
}
