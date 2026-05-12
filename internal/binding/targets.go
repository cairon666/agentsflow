package binding

// Target identifies a supported output tool.
type Target string

const (
	// TargetCodex renders Codex configuration.
	TargetCodex Target = "codex"
	// TargetClaude renders Claude Code configuration.
	TargetClaude Target = "claude"
	// TargetOpenCode renders OpenCode configuration.
	TargetOpenCode Target = "opencode"
)
