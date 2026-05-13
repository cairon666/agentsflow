package schema

import (
	"github.com/cairon666/agentsflow/internal/ir"
	flowtemplate "github.com/cairon666/agentsflow/internal/template"
)

// ToIR converts a validated template into normalized IR.
func ToIR(flow flowtemplate.Flow) ir.Flow {
	out := ir.Flow{
		ID:                 flow.ID,
		Version:            flow.Version,
		ModelSlots:         make(map[string]ir.ModelSlot, len(flow.ModelSlots)),
		PermissionProfiles: make(map[string]ir.PermissionProfile, len(flow.PermissionProfiles)),
		Agents:             make(map[string]ir.Agent, len(flow.Agents)),
		Instructions:       flow.Instructions,
		ToolConfigs:        flow.ToolConfigs,
	}
	for name, slot := range flow.ModelSlots {
		out.ModelSlots[name] = ir.ModelSlot{Description: slot.Description, Fallback: slot.Fallback}
	}
	for name, profile := range flow.PermissionProfiles {
		out.PermissionProfiles[name] = ir.PermissionProfile{
			Description:  profile.Description,
			Capabilities: profile.Capabilities,
		}
	}
	for name, agent := range flow.Agents {
		out.Agents[name] = ir.Agent{
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
