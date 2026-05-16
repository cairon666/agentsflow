package flow

// Flow is the normalized representation used by target renderers.
type Flow struct {
	ID                 string
	Version            int
	ModelSlots         map[string]ModelSlot
	PermissionProfiles map[string]PermissionProfile
	Agents             map[string]Agent
	Instructions       map[string]string
	ToolConfigs        map[string]map[string]any
}

// ResolveAgentModel returns the selected model for agent, following fallback slots.
func (f Flow) ResolveAgentModel(models map[string]string, agent Agent) string {
	if model := models[agent.ModelSlot]; model != "" {
		return model
	}
	seen := map[string]struct{}{agent.ModelSlot: {}}
	for next := f.ModelSlots[agent.ModelSlot].Fallback; next != ""; next = f.ModelSlots[next].Fallback {
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
