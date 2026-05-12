package schema

// CapabilityValues contains the accepted policy values.
var CapabilityValues = map[string]struct{}{
	"allow": {},
	"ask":   {},
	"deny":  {},
}

// Capabilities contains the portable capability vocabulary.
var Capabilities = map[string]struct{}{
	"read_files":       {},
	"list_files":       {},
	"search_code":      {},
	"inspect_metadata": {},
	"edit_files":       {},
	"run_shell":        {},
	"fetch_urls":       {},
	"web_search":       {},
	"spawn_agents":     {},
	"ask_user":         {},
}

// IsCapability reports whether name is a known portable capability.
func IsCapability(name string) bool {
	_, ok := Capabilities[name]
	return ok
}

// IsCapabilityValue reports whether value is a valid capability decision.
func IsCapabilityValue(value string) bool {
	_, ok := CapabilityValues[value]
	return ok
}
