package opencode

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	"github.com/cairon666/agentsflow/internal/exporter"
	"github.com/cairon666/agentsflow/internal/filesystem"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

// Exporter reads OpenCode configuration and exports an agentsflow template.
type Exporter struct {
	Reader filesystem.Reader
}

// NewExporter creates the OpenCode source exporter.
func NewExporter() exporter.Exporter {
	return Exporter{}
}

// Metadata describes the OpenCode source exporter.
func (e Exporter) Metadata() exporter.Metadata {
	return exporter.Metadata{
		Name:    binding.TargetOpenCode,
		Aliases: []string{"opencode", "open-code"},
		Scopes:  exporter.ProjectAndGlobalScopes(),
	}
}

// Export converts OpenCode config files into an agentsflow template spec.
func (e Exporter) Export(_ context.Context, input exporter.ExportInput) (exporter.ExportResult, error) {
	if diags := exporter.ValidateSupportedScope(e.Metadata(), input.Scope); diagnostic.HasErrors(diags) {
		return exporter.ExportResult{Diagnostics: diags}, fmt.Errorf("unsupported export scope")
	}
	root := opencodeRoot(input.Scope, input.WorkDir, input.HomeDir)
	agentsDir := filepath.Join(root, ".opencode", "agents")
	if input.Scope == binding.ScopeGlobal {
		agentsDir = filepath.Join(root, "agents")
	}

	var diags []diagnostic.Diagnostic
	instructions, err := filesystem.ReadOptionalFile(e.Reader, filepath.Join(root, "AGENTS.md"))
	if err != nil {
		return exporter.ExportResult{}, fmt.Errorf("read OpenCode instructions: %w", err)
	}
	instructionMap := map[string]string{}
	if len(instructions) > 0 {
		instructionMap["AGENTS.md"] = string(instructions)
	} else {
		diags = append(diags, diagnostic.Warningf("OpenCode shared instructions file is missing"))
	}

	agentFiles, err := exporter.FilesWithExtension(agentsDir, ".md")
	if err != nil {
		return exporter.ExportResult{}, fmt.Errorf("read OpenCode agents directory: %w", err)
	}
	if len(agentFiles) == 0 {
		return exporter.ExportResult{}, fmt.Errorf("no OpenCode agents found")
	}

	agentIDs := exporter.NewUniqueIDs()
	modelSlotIDs := exporter.NewUniqueIDs()
	permissionProfileIDs := exporter.NewUniqueIDs()
	spec := flowmodel.Spec{
		ID:      fmt.Sprintf("exported-opencode-%s", input.Scope),
		Version: 1,
		ModelSlots: map[string]flowmodel.SpecModelSlot{
			flowmodel.MainModelSlot: {
				Description: "Main model exported from OpenCode.",
			},
		},
		PermissionProfiles: map[string]flowmodel.SpecPermissionProfile{},
		Agents:             map[string]flowmodel.SpecAgent{},
		Instructions:       instructionMap,
	}

	for _, path := range agentFiles {
		frontmatter, body, err := e.readAgent(path)
		if err != nil {
			return exporter.ExportResult{}, err
		}
		nativeID := exporter.StringValue(frontmatter, "name")
		if nativeID == "" {
			nativeID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		id := agentIDs.Next(nativeID, "agent")
		if id != nativeID {
			diags = append(diags, diagnostic.Warningf("OpenCode agent %q was exported as %q", nativeID, id))
		}

		description := exporter.StringValue(frontmatter, "description")
		if description == "" {
			description = fmt.Sprintf("Imported OpenCode agent %s.", id)
			diags = append(diags, diagnostic.Warningf("OpenCode agent %q has no description; generated a fallback", nativeID))
		}
		if strings.TrimSpace(body) == "" {
			return exporter.ExportResult{}, fmt.Errorf("opencode agent %q has empty prompt body", nativeID)
		}

		permissionProfile := permissionProfileIDs.Next(id+"-permissions", "agent-permissions")
		spec.PermissionProfiles[permissionProfile] = exporter.ExportedPermissionProfile("OpenCode", id, opencodeCapabilities(frontmatter))
		modelSlot := modelSlotIDs.Next(id+"-model", "agent-model")
		spec.ModelSlots[modelSlot] = flowmodel.SpecModelSlot{
			Description: fmt.Sprintf("Model for exported OpenCode agent %s.", id),
		}
		spec.Agents[id] = flowmodel.SpecAgent{
			Description:       description,
			ModelSlot:         modelSlot,
			ReasoningEffort:   defaultReasoning(exporter.StringValue(frontmatter, "reasoningEffort")),
			PermissionProfile: permissionProfile,
			Prompt:            body,
		}
	}
	return exporter.ExportResult{Spec: spec, Diagnostics: diags}, nil
}

func (e Exporter) readAgent(path string) (map[string]any, string, error) {
	data, err := filesystem.ReadOptionalFile(e.Reader, path)
	if err != nil {
		return nil, "", fmt.Errorf("read OpenCode agent %q: %w", path, err)
	}
	frontmatter, body, err := exporter.SplitFrontmatter(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse OpenCode agent %q: %w", path, err)
	}
	return frontmatter, body, nil
}

func opencodeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".config", "opencode")
	}
	return workDir
}

func opencodeCapabilities(frontmatter map[string]any) map[string]string {
	permission := exporter.StringMapValue(frontmatter, "permission")
	overrides := map[string]string{}
	mapPermission(permission, "edit", "edit_files", overrides)
	mapPermission(permission, "bash", "run_shell", overrides)
	mapPermission(permission, "webfetch", "fetch_urls", overrides)
	mapPermission(permission, "websearch", "web_search", overrides)
	mapPermission(permission, "task", "spawn_agents", overrides)
	return exporter.FullCapabilities(overrides)
}

func mapPermission(permission map[string]string, nativeKey, capability string, out map[string]string) {
	value := strings.ToLower(strings.TrimSpace(permission[nativeKey]))
	if flowmodel.IsCapabilityValue(value) {
		out[capability] = value
	}
}

func defaultReasoning(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "medium"
	}
	return value
}
