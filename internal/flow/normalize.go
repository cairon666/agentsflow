package flow

// Normalize converts a validated template spec into normalized flow.
func Normalize(spec Spec) Flow {
	modelSlots := normalizeModelSlots(spec.ModelSlots)
	out := Flow{
		ID:                 spec.ID,
		Version:            spec.Version,
		ModelSlots:         modelSlots,
		PermissionProfiles: make(map[string]PermissionProfile, len(spec.PermissionProfiles)),
		Agents:             make(map[string]Agent, len(spec.Agents)),
		Instructions:       spec.Instructions,
	}
	for name, profile := range spec.PermissionProfiles {
		out.PermissionProfiles[name] = PermissionProfile(profile)
	}
	for name, agent := range spec.Agents {
		modelSlot := agent.ModelSlot
		if modelSlot == "" {
			modelSlot = MainModelSlot
		}
		out.Agents[name] = Agent{
			ID:                name,
			Description:       agent.Description,
			ModelSlot:         modelSlot,
			ReasoningEffort:   agent.ReasoningEffort,
			PermissionProfile: agent.PermissionProfile,
			Prompt:            agent.Prompt,
		}
	}
	return out
}

func normalizeModelSlots(slots map[string]SpecModelSlot) map[string]ModelSlot {
	out := make(map[string]ModelSlot, len(slots)+1)
	out[MainModelSlot] = ModelSlot{}
	for name, slot := range slots {
		out[name] = ModelSlot(slot)
	}
	return out
}
