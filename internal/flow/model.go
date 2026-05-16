package flow

// MainModelSlot is always available as the default model slot.
const MainModelSlot = "main"

// Flow is the normalized representation used by target renderers.
type Flow struct {
	ID                 string
	Version            int
	ModelSlots         map[string]ModelSlot
	PermissionProfiles map[string]PermissionProfile
	Agents             map[string]Agent
	Instructions       map[string]string
}

// ResolveAgentModel returns the selected model for agent, following fallback slots.
func (f Flow) ResolveAgentModel(models map[string]string, agent Agent) string {
	slot := agent.ModelSlot
	if slot == "" {
		slot = MainModelSlot
	}
	if model := models[slot]; model != "" {
		return model
	}
	seen := map[string]struct{}{slot: {}}
	for next := f.ModelSlots[slot].Fallback; next != ""; next = f.ModelSlots[next].Fallback {
		if _, ok := seen[next]; ok {
			return ""
		}
		seen[next] = struct{}{}
		if model := models[next]; model != "" {
			return model
		}
	}
	return ""
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
