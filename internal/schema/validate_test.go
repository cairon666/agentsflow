package schema

import (
	"testing"

	"github.com/cairon666/agentflow/internal/diagnostic"
	flowtemplate "github.com/cairon666/agentflow/internal/template"
)

func TestValidateAcceptsMinimalFlow(t *testing.T) {
	flow := minimalFlow()
	diags := Validate(flow)
	if diagnostic.HasErrors(diags) {
		t.Fatalf("expected no errors, got %v", diags)
	}
}

func TestValidateRejectsUnknownCapability(t *testing.T) {
	flow := minimalFlow()
	flow.PermissionProfiles["read"].Capabilities["unknown"] = "allow"
	diags := Validate(flow)
	if !diagnostic.HasErrors(diags) {
		t.Fatalf("expected validation error, got %v", diags)
	}
}

func TestValidateRejectsMissingModelSlot(t *testing.T) {
	flow := minimalFlow()
	agent := flow.Agents["reviewer"]
	agent.ModelSlot = "missing"
	flow.Agents["reviewer"] = agent
	diags := Validate(flow)
	if !diagnostic.HasErrors(diags) {
		t.Fatalf("expected validation error, got %v", diags)
	}
}

func minimalFlow() flowtemplate.Flow {
	return flowtemplate.Flow{
		ID:      "test-flow",
		Version: 1,
		ModelSlots: map[string]flowtemplate.ModelSlot{
			"main": {Description: "Main model"},
		},
		PermissionProfiles: map[string]flowtemplate.PermissionProfile{
			"read": {
				Description: "Read only",
				Capabilities: map[string]string{
					"read_files": "allow",
					"edit_files": "deny",
				},
			},
		},
		Agents: map[string]flowtemplate.Agent{
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
