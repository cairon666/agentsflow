package codex

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/exporter"
	"github.com/cairon666/agentsflow/internal/filesystem"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

// Exporter reads Codex configuration and exports an agentsflow template.
type Exporter struct {
	Reader filesystem.Reader
}

// NewExporter creates the Codex source exporter.
func NewExporter() exporter.Exporter {
	return Exporter{}
}

// Metadata describes the Codex source exporter.
func (e Exporter) Metadata() exporter.Metadata {
	return exporter.Metadata{
		Name:    binding.TargetCodex,
		Aliases: []string{"codex", "openai-codex"},
		Scopes:  exporter.ProjectAndGlobalScopes(),
	}
}

// Export converts Codex config files into an agentsflow template spec.
func (e Exporter) Export(_ context.Context, input exporter.ExportInput) (exporter.ExportResult, error) {
	if diags := exporter.ValidateSupportedScope(e.Metadata(), input.Scope); diagnostic.HasErrors(diags) {
		return exporter.ExportResult{Diagnostics: diags}, fmt.Errorf("unsupported export scope")
	}
	root := codexRoot(input.Scope, input.WorkDir, input.HomeDir)
	configDir := root
	if input.Scope == binding.ScopeProject {
		configDir = filepath.Join(root, ".codex")
	}

	var diags []diagnostic.Diagnostic
	instructions, err := filesystem.ReadOptionalFile(e.Reader, filepath.Join(root, "AGENTS.md"))
	if err != nil {
		return exporter.ExportResult{}, fmt.Errorf("read Codex instructions: %w", err)
	}
	instructionMap := map[string]string{}
	if len(instructions) > 0 {
		instructionMap["AGENTS.md"] = string(instructions)
	} else {
		diags = append(diags, diagnostic.Warningf("Codex shared instructions file is missing"))
	}

	agentFiles, err := exporter.FilesWithExtension(filepath.Join(configDir, "agents"), ".toml")
	if err != nil {
		return exporter.ExportResult{}, fmt.Errorf("read Codex agents directory: %w", err)
	}
	if len(agentFiles) == 0 {
		return exporter.ExportResult{}, fmt.Errorf("no Codex agents found")
	}

	agentIDs := exporter.NewUniqueIDs()
	modelSlotIDs := exporter.NewUniqueIDs()
	permissionProfileIDs := exporter.NewUniqueIDs()
	spec := flowmodel.Spec{
		ID:      fmt.Sprintf("exported-codex-%s", input.Scope),
		Version: 1,
		ModelSlots: map[string]flowmodel.SpecModelSlot{
			flowmodel.MainModelSlot: {
				Description: "Main model exported from Codex.",
			},
		},
		PermissionProfiles: map[string]flowmodel.SpecPermissionProfile{},
		Agents:             map[string]flowmodel.SpecAgent{},
		Instructions:       instructionMap,
	}

	for _, path := range agentFiles {
		agent, err := e.readAgent(path)
		if err != nil {
			return exporter.ExportResult{}, err
		}
		nativeID := strings.TrimSpace(agent.Name)
		if nativeID == "" {
			nativeID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		id := agentIDs.Next(nativeID, "agent")
		if id != nativeID {
			diags = append(diags, diagnostic.Warningf("Codex agent %q was exported as %q", nativeID, id))
		}

		description := strings.TrimSpace(agent.Description)
		if description == "" {
			description = fmt.Sprintf("Imported Codex agent %s.", id)
			diags = append(diags, diagnostic.Warningf("Codex agent %q has no description; generated a fallback", nativeID))
		}
		if strings.TrimSpace(agent.DeveloperInstructions) == "" {
			return exporter.ExportResult{}, fmt.Errorf("codex agent %q has empty developer_instructions", nativeID)
		}

		permissionProfile := permissionProfileIDs.Next(id+"-permissions", "agent-permissions")
		spec.PermissionProfiles[permissionProfile] = exporter.ExportedPermissionProfile("Codex", id, codexCapabilities(agent))
		modelSlot := modelSlotIDs.Next(id+"-model", "agent-model")
		spec.ModelSlots[modelSlot] = flowmodel.SpecModelSlot{
			Description: fmt.Sprintf("Model for exported Codex agent %s.", id),
		}
		spec.Agents[id] = flowmodel.SpecAgent{
			Description:       description,
			ModelSlot:         modelSlot,
			ReasoningEffort:   defaultReasoning(agent.ModelReasoningEffort),
			PermissionProfile: permissionProfile,
			Prompt:            agent.DeveloperInstructions,
		}
	}
	return exporter.ExportResult{Spec: spec, Diagnostics: diags}, nil
}

func (e Exporter) readAgent(path string) (codexAgentConfig, error) {
	data, err := filesystem.ReadOptionalFile(e.Reader, path)
	if err != nil {
		return codexAgentConfig{}, fmt.Errorf("read Codex agent %q: %w", path, err)
	}
	var agent codexAgentConfig
	if err := toml.Unmarshal(data, &agent); err != nil {
		return codexAgentConfig{}, fmt.Errorf("parse Codex agent %q: %w", path, err)
	}
	return agent, nil
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
	DeveloperInstructions string          `toml:"developer_instructions"`
	Model                 string          `toml:"model"`
	ModelReasoningEffort  string          `toml:"model_reasoning_effort"`
	SandboxMode           string          `toml:"sandbox_mode"`
	ApprovalPolicy        string          `toml:"approval_policy"`
	WebSearch             string          `toml:"web_search"`
	SandboxWorkspaceWrite map[string]bool `toml:"sandbox_workspace_write"`
}

func codexCapabilities(agent codexAgentConfig) map[string]string {
	overrides := map[string]string{}
	if strings.EqualFold(agent.SandboxMode, "workspace-write") {
		overrides["edit_files"] = "allow"
		if agent.ApprovalPolicy == "" || !strings.EqualFold(agent.ApprovalPolicy, "never") {
			overrides["run_shell"] = "ask"
		}
	}
	if strings.EqualFold(agent.WebSearch, "live") {
		overrides["web_search"] = "allow"
	}
	if agent.SandboxWorkspaceWrite["network_access"] {
		overrides["fetch_urls"] = "allow"
	}
	return exporter.FullCapabilities(overrides)
}

func defaultReasoning(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "medium"
	}
	return value
}
