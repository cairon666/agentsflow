package flow

// Normalize converts a validated template spec into normalized flow.
func Normalize(spec Spec) Flow {
	out := Flow{
		ID:                 spec.ID,
		Version:            spec.Version,
		ModelSlots:         make(map[string]ModelSlot, len(spec.ModelSlots)),
		PermissionProfiles: make(map[string]PermissionProfile, len(spec.PermissionProfiles)),
		Agents:             make(map[string]Agent, len(spec.Agents)),
		Instructions:       spec.Instructions,
		ToolConfigs:        spec.ToolConfigs,
	}
	for name, slot := range spec.ModelSlots {
		out.ModelSlots[name] = ModelSlot(slot)
	}
	for name, profile := range spec.PermissionProfiles {
		out.PermissionProfiles[name] = PermissionProfile(profile)
	}
	for name, agent := range spec.Agents {
		out.Agents[name] = Agent{
			ID:                name,
			Description:       agent.Description,
			ModelSlot:         agent.ModelSlot,
			ReasoningEffort:   agent.ReasoningEffort,
			PermissionProfile: agent.PermissionProfile,
			Prompt:            agent.Prompt,
		}
	}
	return out
}
