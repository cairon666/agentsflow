package flow

import (
	"fmt"
	"regexp"

	"github.com/cairon666/agentsflow/internal/diagnostic"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ValidateSpec checks the decoded template before it is normalized.
func ValidateSpec(spec Spec) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	if spec.ID == "" {
		diags = append(diags, diagnostic.Errorf("id is required"))
	} else if !idPattern.MatchString(spec.ID) {
		diags = append(diags, diagnostic.Errorf("id %q must contain only letters, numbers, underscores, and hyphens", spec.ID))
	}
	if spec.Version != 1 {
		diags = append(diags, diagnostic.Errorf("version %d is not supported; expected 1", spec.Version))
	}
	if len(spec.PermissionProfiles) == 0 {
		diags = append(diags, diagnostic.Errorf("permission_profiles must contain at least one profile"))
	}
	if len(spec.Agents) == 0 {
		diags = append(diags, diagnostic.Errorf("agents must contain at least one agent"))
	}
	modelSlots := normalizeModelSlots(spec.ModelSlots)
	for name, slot := range modelSlots {
		if !idPattern.MatchString(name) {
			diags = append(diags, diagnostic.Errorf("model slot %q has invalid id", name))
		}
		if slot.Fallback != "" {
			if _, ok := modelSlots[slot.Fallback]; !ok {
				diags = append(diags, diagnostic.Errorf("model slot %q fallback %q does not exist", name, slot.Fallback))
			}
		}
	}
	for name, profile := range spec.PermissionProfiles {
		if !idPattern.MatchString(name) {
			diags = append(diags, diagnostic.Errorf("permission profile %q has invalid id", name))
		}
		if len(profile.Capabilities) == 0 {
			diags = append(diags, diagnostic.Errorf("permission profile %q must define capabilities", name))
		}
		for capability, value := range profile.Capabilities {
			if !IsCapability(capability) {
				diags = append(diags, diagnostic.Errorf("permission profile %q uses unknown capability %q", name, capability))
			}
			if !IsCapabilityValue(value) {
				diags = append(diags, diagnostic.Errorf("permission profile %q capability %q has invalid value %q", name, capability, value))
			}
		}
	}
	for name, agent := range spec.Agents {
		if !idPattern.MatchString(name) {
			diags = append(diags, diagnostic.Errorf("agent %q has invalid id", name))
		}
		if agent.Description == "" {
			diags = append(diags, diagnostic.Errorf("agent %q description is required", name))
		}
		if agent.Prompt == "" {
			diags = append(diags, diagnostic.Errorf("agent %q prompt is required", name))
		}
		modelSlot := agent.ModelSlot
		if modelSlot == "" {
			modelSlot = MainModelSlot
		}
		if _, ok := modelSlots[modelSlot]; !ok {
			diags = append(diags, diagnostic.Errorf("agent %q references missing model_slot %q", name, modelSlot))
		}
		if _, ok := spec.PermissionProfiles[agent.PermissionProfile]; !ok {
			diags = append(diags, diagnostic.Errorf("agent %q references missing permission_profile %q", name, agent.PermissionProfile))
		}
	}
	if _, ok := spec.Instructions["AGENTS.md"]; !ok {
		diags = append(diags, diagnostic.Warningf("instructions.AGENTS.md is not defined; generated configs may lack shared project instructions"))
	}
	for name := range spec.Instructions {
		if name == "" {
			diags = append(diags, diagnostic.Errorf("instructions contains an empty filename"))
		}
		if name == "." || name == ".." {
			diags = append(diags, diagnostic.Errorf("instructions contains invalid filename %q", name))
		}
	}
	return dedupe(diags)
}

func dedupe(diags []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	seen := make(map[string]struct{}, len(diags))
	out := make([]diagnostic.Diagnostic, 0, len(diags))
	for _, diag := range diags {
		key := fmt.Sprintf("%s:%s", diag.Severity, diag.Message)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, diag)
	}
	return out
}
