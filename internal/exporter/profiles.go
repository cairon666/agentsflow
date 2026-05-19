package exporter

import (
	"fmt"

	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

// BaseCapabilities returns a complete read-only capability map.
func BaseCapabilities() map[string]string {
	return map[string]string{
		"read_files":       "allow",
		"list_files":       "allow",
		"search_code":      "allow",
		"inspect_metadata": "allow",
		"edit_files":       "deny",
		"run_shell":        "deny",
		"fetch_urls":       "deny",
		"web_search":       "deny",
		"spawn_agents":     "deny",
		"ask_user":         "allow",
	}
}

// FullCapabilities overlays native permission hints onto a complete capability map.
func FullCapabilities(overrides map[string]string) map[string]string {
	caps := BaseCapabilities()
	for key, value := range overrides {
		if flowmodel.IsCapability(key) && flowmodel.IsCapabilityValue(value) {
			caps[key] = value
		}
	}
	return caps
}

// ExportedPermissionProfile creates a per-agent template permission profile.
func ExportedPermissionProfile(source, agentID string, caps map[string]string) flowmodel.SpecPermissionProfile {
	return flowmodel.SpecPermissionProfile{
		Description:  fmt.Sprintf("Permissions exported from %s for agent %s.", source, agentID),
		Capabilities: FullCapabilities(caps),
	}
}
