package opencode

import (
	"context"
	"path/filepath"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/render"
	"github.com/cairon666/agentsflow/internal/target"
)

// Renderer renders OpenCode configuration.
type Renderer struct{}

// New creates the OpenCode target renderer.
func New() target.Renderer {
	return Renderer{}
}

// Metadata describes the OpenCode target renderer.
func (r Renderer) Metadata() target.Metadata {
	return target.Metadata{
		Name:    binding.TargetOpenCode,
		Aliases: []string{"opencode", "open-code"},
		Scopes:  target.ProjectAndGlobalScopes(),
	}
}

func (r Renderer) Validate(_ context.Context, input target.RenderInput) []diagnostic.Diagnostic {
	return target.ValidateSupportedScope(r.Metadata(), input.Scope)
}

func (r Renderer) Render(_ context.Context, input target.RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic) {
	root := opencodeRoot(input.Scope, input.WorkDir, input.HomeDir)
	agentsDir := filepath.Join(root, ".opencode", "agents")
	if input.Scope == binding.ScopeGlobal {
		agentsDir = filepath.Join(root, "agents")
	}
	files := []install.DesiredFile{}
	addDesired := func(path string, content []byte, strategy install.FileStrategy) {
		files = append(files, install.DesiredFile{Path: path, Content: content, Strategy: strategy})
	}
	if agents, ok := input.Flow.Instructions["AGENTS.md"]; ok {
		addDesired(filepath.Join(root, "AGENTS.md"), []byte(agents), install.StrategyCreateOnly)
	}
	config, err := render.JSON(map[string]any{"model": input.Models["main"]})
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(filepath.Join(root, "opencode.json"), config, install.StrategyOwned)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		body := render.Frontmatter(opencodeFrontmatter(agent, profile, input.Flow.ResolveAgentModel(input.Models, agent))) + agent.Prompt
		addDesired(filepath.Join(agentsDir, id+".md"), []byte(body), install.StrategyOwned)
	}
	return install.ArtifactSet{Target: string(r.Metadata().Name), Scope: string(input.Scope), Files: files}, nil
}

func opencodeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".config", "opencode")
	}
	return workDir
}

func opencodeFrontmatter(agent flowmodel.Agent, profile flowmodel.PermissionProfile, model string) map[string]any {
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
