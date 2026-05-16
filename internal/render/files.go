package render

import (
	"sort"
	"strings"

	"github.com/cairon666/agentsflow/internal/ir"
)

// AgentIDs returns sorted agent ids.
func AgentIDs(flow ir.Flow) []string {
	ids := make([]string, 0, len(flow.Agents))
	for id := range flow.Agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HyphenID returns an id suitable for tools that prefer hyphens.
func HyphenID(id string) string {
	return strings.ReplaceAll(id, "_", "-")
}

// ModelFor returns the resolved model for an agent.
func ModelFor(agent ir.Agent, models map[string]string, fallbacks map[string]string) string {
	if model := models[agent.ModelSlot]; model != "" {
		return model
	}
	seen := map[string]struct{}{agent.ModelSlot: {}}
	for next := fallbacks[agent.ModelSlot]; next != ""; next = fallbacks[next] {
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

// Fallbacks returns slot fallback lookup.
func Fallbacks(flow ir.Flow) map[string]string {
	out := make(map[string]string, len(flow.ModelSlots))
	for name, slot := range flow.ModelSlots {
		out[name] = slot.Fallback
	}
	return out
}
