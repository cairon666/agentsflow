package opencode

import (
	"context"
	"path/filepath"

	"github.com/cairon666/agentflow/internal/adapter"
	"github.com/cairon666/agentflow/internal/binding"
	"github.com/cairon666/agentflow/internal/diagnostic"
	"github.com/cairon666/agentflow/internal/install"
	"github.com/cairon666/agentflow/internal/ir"
	"github.com/cairon666/agentflow/internal/render"
)

// Adapter renders OpenCode configuration.
type Adapter struct{}

func (a Adapter) Target() binding.Target { return binding.TargetOpenCode }

func (a Adapter) Aliases() []string { return []string{"opencode", "open-code"} }

func (a Adapter) Validate(_ context.Context, _ ir.Flow) []diagnostic.Diagnostic { return nil }

func (a Adapter) Render(_ context.Context, input adapter.RenderInput) (install.Plan, []diagnostic.Diagnostic) {
	root := opencodeRoot(input.Scope, input.WorkDir, input.HomeDir)
	agentsDir := filepath.Join(root, ".opencode", "agents")
	if input.Scope == binding.ScopeGlobal {
		agentsDir = filepath.Join(root, "agents")
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
	config, err := render.JSON(map[string]any{"model": input.Models["main"]})
	if err != nil {
		return install.Plan{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(filepath.Join(root, "opencode.json"), config)
	fallbacks := render.Fallbacks(input.Flow)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		body := render.Frontmatter(opencodeFrontmatter(agent, profile, render.ModelFor(agent, input.Models, fallbacks))) + agent.Prompt
		addDesired(filepath.Join(agentsDir, id+".md"), []byte(body))
	}
	return install.BuildPlanWithManagedPaths(string(a.Target()), string(input.Scope), desired, managedPaths), nil
}

func opencodeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".config", "opencode")
	}
	return workDir
}

func opencodeFrontmatter(agent ir.Agent, profile ir.PermissionProfile, model string) map[string]any {
	permission := map[string]string{
		"edit":      profile.Capabilities["edit_files"],
		"bash":      profile.Capabilities["run_shell"],
		"webfetch":  profile.Capabilities["fetch_urls"],
		"websearch": profile.Capabilities["web_search"],
		"task":      profile.Capabilities["spawn_agents"],
	}
	return map[string]any{
		"description":     agent.Description,
		"mode":            "subagent",
		"model":           model,
		"reasoningEffort": agent.ReasoningEffort,
		"permission":      permission,
	}
}
