package template

// Flow is the YAML template shape.
type Flow struct {
	ID                 string                       `yaml:"id"`
	Version            int                          `yaml:"version"`
	ModelSlots         map[string]ModelSlot         `yaml:"model_slots"`
	PermissionProfiles map[string]PermissionProfile `yaml:"permission_profiles"`
	Agents             map[string]Agent             `yaml:"agents"`
	Instructions       map[string]string            `yaml:"instructions"`
	ToolConfigs        map[string]map[string]any    `yaml:"tool_configs"`
}

// ModelSlot describes a logical model binding requested from the user.
type ModelSlot struct {
	Description string `yaml:"description"`
	Fallback    string `yaml:"fallback"`
}

// PermissionProfile binds capabilities to allow, ask, or deny decisions.
type PermissionProfile struct {
	Description  string            `yaml:"description"`
	Capabilities map[string]string `yaml:"capabilities"`
}

// Agent describes a reusable subagent role.
type Agent struct {
	Description       string `yaml:"description"`
	ModelSlot         string `yaml:"model_slot"`
	ReasoningEffort   string `yaml:"reasoning_effort"`
	PermissionProfile string `yaml:"permission_profile"`
	Prompt            string `yaml:"prompt"`
}
