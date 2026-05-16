package flow

import "testing"

func TestFlowResolveAgentModel(t *testing.T) {
	flow := Flow{
		ModelSlots: map[string]ModelSlot{
			"main":     {},
			"code":     {Fallback: "main"},
			"review":   {Fallback: "code"},
			"cycle_a":  {Fallback: "cycle_b"},
			"cycle_b":  {Fallback: "cycle_a"},
			"orphaned": {Fallback: "missing"},
		},
	}
	tests := []struct {
		name   string
		models map[string]string
		agent  Agent
		want   string
	}{
		{
			name:   "direct slot model",
			models: map[string]string{"code": "gpt-code", "main": "gpt-main"},
			agent:  Agent{ModelSlot: "code"},
			want:   "gpt-code",
		},
		{
			name:   "fallback slot model",
			models: map[string]string{"main": "gpt-main"},
			agent:  Agent{ModelSlot: "review"},
			want:   "gpt-main",
		},
		{
			name:   "empty slot defaults to main",
			models: map[string]string{"main": "gpt-main"},
			agent:  Agent{},
			want:   "gpt-main",
		},
		{
			name:   "cycle returns empty",
			models: map[string]string{},
			agent:  Agent{ModelSlot: "cycle_a"},
			want:   "",
		},
		{
			name:   "missing fallback returns empty",
			models: map[string]string{},
			agent:  Agent{ModelSlot: "orphaned"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flow.ResolveAgentModel(tt.models, tt.agent)
			if got != tt.want {
				t.Fatalf("model = %q, want %q", got, tt.want)
			}
		})
	}
}
