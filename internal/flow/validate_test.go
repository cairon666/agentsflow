package flow

import (
	"testing"

	"github.com/cairon666/agentsflow/internal/diagnostic"
)

func TestValidateSpecAcceptsMinimalFlow(t *testing.T) {
	spec := minimalSpec()

	diags := ValidateSpec(spec)

	if diagnostic.HasErrors(diags) {
		t.Fatalf("expected no errors, got %v", diags)
	}
}

func TestValidateSpecRejectsUnknownCapability(t *testing.T) {
	spec := minimalSpec()
	spec.PermissionProfiles["read"].Capabilities["unknown"] = "allow"

	diags := ValidateSpec(spec)

	if !diagnostic.HasErrors(diags) {
		t.Fatalf("expected validation error, got %v", diags)
	}
}

func TestValidateSpecRejectsMissingModelSlot(t *testing.T) {
	spec := minimalSpec()
	agent := spec.Agents["reviewer"]
	agent.ModelSlot = "missing"
	spec.Agents["reviewer"] = agent

	diags := ValidateSpec(spec)

	if !diagnostic.HasErrors(diags) {
		t.Fatalf("expected validation error, got %v", diags)
	}
}

func minimalSpec() Spec {
	return Spec{
		ID:      "test-flow",
		Version: 1,
		ModelSlots: map[string]SpecModelSlot{
			"main": {Description: "Main model"},
		},
		PermissionProfiles: map[string]SpecPermissionProfile{
			"read": {
				Description: "Read only",
				Capabilities: map[string]string{
					"read_files": "allow",
					"edit_files": "deny",
				},
			},
		},
		Agents: map[string]SpecAgent{
			"reviewer": {
				Description:       "Reviews code",
				ModelSlot:         "main",
				ReasoningEffort:   "medium",
				PermissionProfile: "read",
				Prompt:            "Review code.",
			},
		},
		Instructions: map[string]string{
			"AGENTS.md": "# Test",
		},
	}
}
