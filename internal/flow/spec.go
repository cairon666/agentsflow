package flow

// Spec is the YAML template shape.
type Spec struct {
	ID                 string                           `yaml:"id"`
	Version            int                              `yaml:"version"`
	ModelSlots         map[string]SpecModelSlot         `yaml:"model_slots"`
	PermissionProfiles map[string]SpecPermissionProfile `yaml:"permission_profiles"`
	Agents             map[string]SpecAgent             `yaml:"agents"`
	Instructions       map[string]string                `yaml:"instructions"`
}

// SpecModelSlot describes a logical model binding requested from the user.
type SpecModelSlot struct {
	Description string `yaml:"description"`
	Fallback    string `yaml:"fallback,omitempty"`
}

// SpecPermissionProfile binds capabilities to allow, ask, or deny decisions.
type SpecPermissionProfile struct {
	Description  string            `yaml:"description"`
	Capabilities map[string]string `yaml:"capabilities"`
}

// SpecAgent describes a reusable subagent role in the template spec.
type SpecAgent struct {
	Description       string `yaml:"description"`
	ModelSlot         string `yaml:"model_slot,omitempty"`
	ReasoningEffort   string `yaml:"reasoning_effort,omitempty"`
	PermissionProfile string `yaml:"permission_profile"`
	Prompt            string `yaml:"prompt"`
}
