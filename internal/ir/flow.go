package ir

// Flow is the normalized representation used by adapters.
type Flow struct {
	ID                 string
	Version            int
	ModelSlots         map[string]ModelSlot
	PermissionProfiles map[string]PermissionProfile
	Agents             map[string]Agent
	Instructions       map[string]string
	ToolConfigs        map[string]map[string]any
}

// ModelSlot describes a logical model binding.
type ModelSlot struct {
	Description string
	Fallback    string
}

// PermissionProfile binds capabilities to allow, ask, or deny.
type PermissionProfile struct {
	Description  string
	Capabilities map[string]string
}

// Agent describes a role rendered into a target-specific agent definition.
type Agent struct {
	ID                string
	Description       string
	ModelSlot         string
	ReasoningEffort   string
	PermissionProfile string
	Prompt            string
}
