package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/ir"
	"github.com/cairon666/agentsflow/internal/render"
)

// Adapter renders Claude Code configuration.
type Adapter struct{}

func (a Adapter) Target() binding.Target { return binding.TargetClaude }

func (a Adapter) Aliases() []string { return []string{"claude", "claude-code", "claudecode"} }

func (a Adapter) Validate(_ context.Context, _ ir.Flow) []diagnostic.Diagnostic { return nil }

func (a Adapter) Render(_ context.Context, input adapter.RenderInput) (install.Plan, []diagnostic.Diagnostic) {
	root := claudeRoot(input.Scope, input.WorkDir, input.HomeDir)
	configDir := root
	if input.Scope == binding.ScopeProject {
		configDir = filepath.Join(root, ".claude")
	}
	desired := map[string][]byte{}
	managedPaths := map[string]struct{}{}
	addDesired := func(path string, content []byte) {
		desired[path] = content
		managedPaths[path] = struct{}{}
	}
	if agents, ok := input.Flow.Instructions["AGENTS.md"]; ok {
		addDesired(filepath.Join(root, "CLAUDE.md"), []byte(agents))
	}
	settingsPath := filepath.Join(configDir, "settings.json")
	settings, err := mergedSettings(settingsPath, input.Models["main"])
	if err != nil {
		return install.Plan{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(settingsPath, settings)
	fallbacks := render.Fallbacks(input.Flow)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		name := render.HyphenID(id)
		body := render.Frontmatter(claudeFrontmatter(name, agent, profile, render.ModelFor(agent, input.Models, fallbacks))) + agent.Prompt
		addDesired(filepath.Join(configDir, "agents", name+".md"), []byte(body))
	}
	return install.BuildPlanWithManagedPaths(string(a.Target()), string(input.Scope), desired, managedPaths), nil
}

func claudeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".claude")
	}
	return workDir
}

func claudeFrontmatter(name string, agent ir.Agent, profile ir.PermissionProfile, model string) map[string]any {
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

func claudeReadOnlyTools(profile ir.PermissionProfile) []string {
	tools := []string{"Read", "Grep", "Glob"}
	if profile.Capabilities["fetch_urls"] == "allow" {
		tools = append(tools, "WebFetch")
	}
	if profile.Capabilities["web_search"] == "allow" {
		tools = append(tools, "WebSearch")
	}
	return tools
}

func mergedSettings(path string, model string) ([]byte, error) {
	settings := map[string]any{}
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return nil, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
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
