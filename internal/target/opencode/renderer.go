package opencode

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/filesystem"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/render"
	"github.com/cairon666/agentsflow/internal/target"
)

// Renderer renders OpenCode configuration.
type Renderer struct {
	Reader filesystem.Reader
}

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
		addDesired(filepath.Join(root, "AGENTS.md"), []byte(agents), install.StrategyOverwrite)
	}
	configPath := filepath.Join(root, "opencode.json")
	existingConfig, err := filesystem.ReadOptionalFile(r.Reader, configPath)
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	config, err := MergeOpenCodeConfig(existingConfig, input.Models["main"])
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(configPath, config, install.StrategyMerge)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		body := render.Frontmatter(opencodeFrontmatter(agent, profile, input.Flow.ResolveAgentModel(input.Models, agent))) + agent.Prompt
		addDesired(filepath.Join(agentsDir, id+".md"), []byte(body), install.StrategyOverwrite)
	}
	return install.ArtifactSet{
		Target:    string(r.Metadata().Name),
		Scope:     string(input.Scope),
		CleanDirs: []string{agentsDir},
		Files:     files,
	}, nil
}

func opencodeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".config", "opencode")
	}
	return workDir
}

// MergeOpenCodeConfig applies agentsflow-managed OpenCode config keys to existing JSON.
func MergeOpenCodeConfig(existing []byte, model string) ([]byte, error) {
	config := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &config); err != nil {
			return nil, err
		}
	}
	config["model"] = model
	return render.JSON(config)
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
