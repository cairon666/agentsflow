package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/ir"
	"github.com/cairon666/agentsflow/internal/render"
)

// Adapter renders Codex configuration.
type Adapter struct{}

func (a Adapter) Target() binding.Target { return binding.TargetCodex }

func (a Adapter) Aliases() []string { return []string{"codex", "openai-codex"} }

func (a Adapter) Validate(_ context.Context, _ ir.Flow) []diagnostic.Diagnostic { return nil }

func (a Adapter) Render(_ context.Context, input adapter.RenderInput) (install.Plan, []diagnostic.Diagnostic) {
	root := codexRoot(input.Scope, input.WorkDir, input.HomeDir)
	configDir := root
	if input.Scope == binding.ScopeProject {
		configDir = filepath.Join(root, ".codex")
	}
	desired := map[string][]byte{}
	managedPaths := map[string]struct{}{}
	addDesired := func(path string, content []byte) {
		desired[path] = content
		managedPaths[path] = struct{}{}
	}
	if agents, ok := input.Flow.Instructions["AGENTS.md"]; ok {
		addDesired(filepath.Join(root, "AGENTS.md"), []byte(agents))
	}
	configPath := filepath.Join(configDir, "config.toml")
	config, err := mergedConfig(configPath, input.Models["main"])
	if err != nil {
		return install.Plan{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(configPath, config)
	fallbacks := render.Fallbacks(input.Flow)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		content, err := render.TOML(codexAgent(agent, profile, render.ModelFor(agent, input.Models, fallbacks)))
		if err != nil {
			return install.Plan{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
		}
		addDesired(filepath.Join(configDir, "agents", id+".toml"), content)
	}
	return install.BuildPlanWithManagedPaths(string(a.Target()), string(input.Scope), desired, managedPaths), nil
}

func mergedConfig(path string, model string) ([]byte, error) {
	config := map[string]any{}
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if err := toml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	config["model"] = model
	config["model_reasoning_effort"] = "high"
	config["plan_mode_reasoning_effort"] = "xhigh"

	features := mapValue(config["features"])
	features["multi_agent"] = true
	config["features"] = features

	agents := mapValue(config["agents"])
	agents["max_threads"] = 7
	agents["max_depth"] = 2
	config["agents"] = agents

	return render.TOML(config)
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func codexRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".codex")
	}
	return workDir
}

type codexAgentConfig struct {
	Name                  string          `toml:"name"`
	Description           string          `toml:"description"`
	DeveloperInstructions string          `toml:"developer_instructions,multiline"`
	Model                 string          `toml:"model"`
	ModelReasoningEffort  string          `toml:"model_reasoning_effort"`
	SandboxMode           string          `toml:"sandbox_mode"`
	ApprovalPolicy        string          `toml:"approval_policy"`
	WebSearch             string          `toml:"web_search"`
	SandboxWorkspaceWrite map[string]bool `toml:"sandbox_workspace_write,omitempty"`
}

func codexAgent(agent ir.Agent, profile ir.PermissionProfile, model string) codexAgentConfig {
	sandboxMode := "read-only"
	approvalPolicy := "never"
	if profile.Capabilities["edit_files"] == "allow" {
		sandboxMode = "workspace-write"
		approvalPolicy = "on-request"
	}
	cfg := codexAgentConfig{
		Name:                  agent.ID,
		Description:           agent.Description,
		DeveloperInstructions: agent.Prompt,
		Model:                 model,
		ModelReasoningEffort:  agent.ReasoningEffort,
		SandboxMode:           sandboxMode,
		ApprovalPolicy:        approvalPolicy,
		WebSearch:             codexWebSearch(profile),
	}
	if sandboxMode == "workspace-write" {
		cfg.SandboxWorkspaceWrite = map[string]bool{"network_access": profile.Capabilities["fetch_urls"] == "allow" || profile.Capabilities["web_search"] == "allow"}
	}
	if model == "" {
		cfg.Model = fmt.Sprintf("{{ models.%s }}", agent.ModelSlot)
	}
	return cfg
}

func codexWebSearch(profile ir.PermissionProfile) string {
	if profile.Capabilities["web_search"] == "allow" {
		return "live"
	}
	return "disabled"
}
