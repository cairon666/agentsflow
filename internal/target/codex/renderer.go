package codex

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/filesystem"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/render"
	"github.com/cairon666/agentsflow/internal/target"
)

// Renderer renders Codex configuration.
type Renderer struct {
	Reader filesystem.Reader
}

// New creates the Codex target renderer.
func New() target.Renderer {
	return Renderer{}
}

// Metadata describes the Codex target renderer.
func (r Renderer) Metadata() target.Metadata {
	return target.Metadata{
		Name:    binding.TargetCodex,
		Aliases: []string{"codex", "openai-codex"},
		Scopes:  target.ProjectAndGlobalScopes(),
	}
}

func (r Renderer) Validate(_ context.Context, input target.RenderInput) []diagnostic.Diagnostic {
	return target.ValidateSupportedScope(r.Metadata(), input.Scope)
}

func (r Renderer) Render(_ context.Context, input target.RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic) {
	root := codexRoot(input.Scope, input.WorkDir, input.HomeDir)
	configDir := root
	if input.Scope == binding.ScopeProject {
		configDir = filepath.Join(root, ".codex")
	}
	files := []install.DesiredFile{}
	addDesired := func(path string, content []byte, strategy install.FileStrategy) {
		files = append(files, install.DesiredFile{Path: path, Content: content, Strategy: strategy})
	}
	if agents, ok := input.Flow.Instructions["AGENTS.md"]; ok {
		addDesired(filepath.Join(root, "AGENTS.md"), []byte(agents), install.StrategyCreateOnly)
	}
	configPath := filepath.Join(configDir, "config.toml")
	existingConfig, err := filesystem.ReadOptionalFile(r.Reader, configPath)
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	config, err := MergeCodexConfig(existingConfig, input.Models["main"])
	if err != nil {
		return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
	}
	addDesired(configPath, config, install.StrategyMerge)
	for _, id := range render.AgentIDs(input.Flow) {
		agent := input.Flow.Agents[id]
		profile := input.Flow.PermissionProfiles[agent.PermissionProfile]
		content, err := render.TOML(codexAgent(agent, profile, input.Flow.ResolveAgentModel(input.Models, agent)))
		if err != nil {
			return install.ArtifactSet{}, []diagnostic.Diagnostic{diagnostic.Errorf("%s", err.Error())}
		}
		addDesired(filepath.Join(configDir, "agents", id+".toml"), content, install.StrategyOwned)
	}
	return install.ArtifactSet{Target: string(r.Metadata().Name), Scope: string(input.Scope), Files: files}, nil
}

// MergeCodexConfig applies agentsflow-managed Codex config keys to existing TOML.
func MergeCodexConfig(existing []byte, model string) ([]byte, error) {
	config := map[string]any{}
	if len(existing) > 0 {
		if err := toml.Unmarshal(existing, &config); err != nil {
			return nil, err
		}
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

func codexAgent(agent flowmodel.Agent, profile flowmodel.PermissionProfile, model string) codexAgentConfig {
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

func codexWebSearch(profile flowmodel.PermissionProfile) string {
	if profile.Capabilities["web_search"] == "allow" {
		return "live"
	}
	return "disabled"
}
