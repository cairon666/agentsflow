package claude

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

// Renderer renders Claude Code configuration.
type Renderer struct {
	Reader filesystem.Reader
}

// New creates the Claude Code target renderer.
func New() target.Renderer {
	return Renderer{}
}

// Metadata describes the Claude Code target renderer.
func (r Renderer) Metadata() target.Metadata {
	return target.Metadata{
		Name:    binding.TargetClaude,
		Aliases: []string{"claude", "claude-code", "claudecode"},
		Scopes:  target.ProjectAndGlobalScopes(),
	}
}

func (r Renderer) Validate(_ context.Context, input target.RenderInput) []diagnostic.Diagnostic {
	return target.ValidateSupportedScope(r.Metadata(), input.Scope)
}

func (r Renderer) Render(_ context.Context, input target.RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic) {
	root := claudeRoot(input.Scope, input.WorkDir, input.HomeDir)
	configDir := root
	if input.Scope == binding.ScopeProject {
		configDir = filepath.Join(root, ".claude")
	}
	files := []install.DesiredFile{}
	addDesired := func(path string, content []byte, strategy install.FileStrategy) {
		files = append(files, install.DesiredFile{Path: path, Content: content, Strategy: strategy})
	}
	if agents, ok := input.Flow.Instructions["AGENTS.md"]; ok {
		addDesired(filepath.Join(root, "CLAUDE.md"), []byte(agents), install.StrategyCreateOnly)
	}
	settingsPath := filepath.Join(configDir, "settings.json")
	existingSettings, err := filesystem.ReadOptionalFile(r.Reader, settingsPath)
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	settings, err := MergeClaudeSettings(existingSettings, input.Models["main"])
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(settingsPath, settings, install.StrategyMerge)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		name := render.HyphenID(id)
		body := render.Frontmatter(claudeFrontmatter(name, agent, profile, input.Flow.ResolveAgentModel(input.Models, agent))) + agent.Prompt
		addDesired(filepath.Join(configDir, "agents", name+".md"), []byte(body), install.StrategyOwned)
	}
	return install.ArtifactSet{Target: string(r.Metadata().Name), Scope: string(input.Scope), Files: files}, nil
}

func claudeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".claude")
	}
	return workDir
}

func claudeFrontmatter(name string, agent flowmodel.Agent, profile flowmodel.PermissionProfile, model string) map[string]any {
	values := map[string]any{
		"name":        name,
		"description": agent.Description,
		"model":       model,
		"effort":      agent.ReasoningEffort,
	}
	if profile.Capabilities["edit_files"] == "deny" {
		values["tools"] = claudeReadOnlyTools(profile)
		values["disallowedTools"] = []string{"Edit", "Write"}
		values["permissionMode"] = "plan"
	}
	return values
}

func claudeReadOnlyTools(profile flowmodel.PermissionProfile) []string {
	tools := []string{"Read", "Grep", "Glob"}
	if profile.Capabilities["fetch_urls"] == "allow" {
		tools = append(tools, "WebFetch")
	}
	if profile.Capabilities["web_search"] == "allow" {
		tools = append(tools, "WebSearch")
	}
	return tools
}

// MergeClaudeSettings applies agentsflow-managed Claude settings to existing JSON.
func MergeClaudeSettings(existing []byte, model string) ([]byte, error) {
	settings := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return nil, err
		}
	}
	settings["alwaysThinkingEnabled"] = true
	settings["model"] = model
	features, _ := settings["features"].(map[string]any)
	if features == nil {
		features = map[string]any{}
	}
	features["multiAgent"] = true
	settings["features"] = features
	return render.JSON(settings)
}
