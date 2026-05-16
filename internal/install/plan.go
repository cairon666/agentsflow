package install

// ActionKind describes a planned filesystem operation.
type ActionKind string

const (
	// ActionCreate creates a new file.
	ActionCreate ActionKind = "create"
	// ActionCleanDir removes all contents from an existing directory before writing files.
	ActionCleanDir ActionKind = "clean-dir"
	// ActionUpdate updates a file allowed by its strategy.
	ActionUpdate ActionKind = "update"
	// ActionOverwrite replaces an existing file allowed by its strategy.
	ActionOverwrite ActionKind = "overwrite"
	// ActionSkip leaves an identical file untouched.
	ActionSkip ActionKind = "skip"
	// ActionConflict reports an unsafe existing file.
	ActionConflict ActionKind = "conflict"
)

// FileStrategy describes how a desired file may interact with an existing file.
type FileStrategy string

const (
	// StrategyMerge updates files after target-specific merge logic preserved user keys.
	StrategyMerge FileStrategy = "merge"
	// StrategyOverwrite replaces files that are fully managed by agentsflow.
	StrategyOverwrite FileStrategy = "overwrite"
	// StrategyCreateOnly creates missing files and conflicts on differing existing files.
	StrategyCreateOnly FileStrategy = "create-only"
)

// Action is one planned filesystem operation.
type Action struct {
	Path            string
	Kind            ActionKind
	Content         []byte
	ExistingContent []byte
	Strategy        FileStrategy
}

// DesiredFile is a target-rendered file before install planning.
type DesiredFile struct {
	Path     string
	Content  []byte
	Strategy FileStrategy
}

// ArtifactSet is the set of target-rendered files to install.
type ArtifactSet struct {
	Target    string
	Scope     string
	CleanDirs []string
	Files     []DesiredFile
}

// Plan is the set of files a target renderer wants to install.
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
