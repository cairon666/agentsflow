package claude

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

// Exporter reads Claude Code configuration and exports an agentsflow template.
type Exporter struct {
	Reader filesystem.Reader
}

// NewExporter creates the Claude Code source exporter.
func NewExporter() exporter.Exporter {
	return Exporter{}
}

// Metadata describes the Claude Code source exporter.
func (e Exporter) Metadata() exporter.Metadata {
	return exporter.Metadata{
		Name:    binding.TargetClaude,
		Aliases: []string{"claude", "claude-code", "claudecode"},
		Scopes:  exporter.ProjectAndGlobalScopes(),
	}
}

// Export converts Claude Code config files into an agentsflow template spec.
func (e Exporter) Export(_ context.Context, input exporter.ExportInput) (exporter.ExportResult, error) {
	if diags := exporter.ValidateSupportedScope(e.Metadata(), input.Scope); diagnostic.HasErrors(diags) {
		return exporter.ExportResult{Diagnostics: diags}, fmt.Errorf("unsupported export scope")
	}
	root := claudeRoot(input.Scope, input.WorkDir, input.HomeDir)
	configDir := root
	if input.Scope == binding.ScopeProject {
		configDir = filepath.Join(root, ".claude")
	}

	var diags []diagnostic.Diagnostic
	instructions, err := filesystem.ReadOptionalFile(e.Reader, filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		return exporter.ExportResult{}, fmt.Errorf("read Claude instructions: %w", err)
	}
	instructionMap := map[string]string{}
	if len(instructions) > 0 {
		instructionMap["AGENTS.md"] = string(instructions)
	} else {
		diags = append(diags, diagnostic.Warningf("Claude shared instructions file is missing"))
	}

	agentFiles, err := exporter.FilesWithExtension(filepath.Join(configDir, "agents"), ".md")
	if err != nil {
		return exporter.ExportResult{}, fmt.Errorf("read Claude agents directory: %w", err)
	}
	if len(agentFiles) == 0 {
		return exporter.ExportResult{}, fmt.Errorf("no Claude agents found")
	}

	agentIDs := exporter.NewUniqueIDs()
	modelSlotIDs := exporter.NewUniqueIDs()
	permissionProfileIDs := exporter.NewUniqueIDs()
	spec := flowmodel.Spec{
		ID:      fmt.Sprintf("exported-claude-%s", input.Scope),
		Version: 1,
		ModelSlots: map[string]flowmodel.SpecModelSlot{
			flowmodel.MainModelSlot: {
				Description: "Main model exported from Claude Code.",
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
			diags = append(diags, diagnostic.Warningf("Claude agent %q was exported as %q", nativeID, id))
		}

		description := exporter.StringValue(frontmatter, "description")
		if description == "" {
			description = fmt.Sprintf("Imported Claude Code agent %s.", id)
			diags = append(diags, diagnostic.Warningf("Claude agent %q has no description; generated a fallback", nativeID))
		}
		if strings.TrimSpace(body) == "" {
			return exporter.ExportResult{}, fmt.Errorf("claude agent %q has empty prompt body", nativeID)
		}

		permissionProfile := permissionProfileIDs.Next(id+"-permissions", "agent-permissions")
		spec.PermissionProfiles[permissionProfile] = exporter.ExportedPermissionProfile("Claude Code", id, claudeCapabilities(frontmatter))
		modelSlot := modelSlotIDs.Next(id+"-model", "agent-model")
		spec.ModelSlots[modelSlot] = flowmodel.SpecModelSlot{
			Description: fmt.Sprintf("Model for exported Claude Code agent %s.", id),
		}
		spec.Agents[id] = flowmodel.SpecAgent{
			Description:       description,
			ModelSlot:         modelSlot,
			ReasoningEffort:   defaultReasoning(exporter.StringValue(frontmatter, "effort")),
			PermissionProfile: permissionProfile,
			Prompt:            body,
		}
	}
	return exporter.ExportResult{Spec: spec, Diagnostics: diags}, nil
}

func (e Exporter) readAgent(path string) (map[string]any, string, error) {
	data, err := filesystem.ReadOptionalFile(e.Reader, path)
	if err != nil {
		return nil, "", fmt.Errorf("read Claude agent %q: %w", path, err)
	}
	frontmatter, body, err := exporter.SplitFrontmatter(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse Claude agent %q: %w", path, err)
	}
	return frontmatter, body, nil
}

func claudeRoot(scope binding.Scope, workDir, homeDir string) string {
	if scope == binding.ScopeGlobal {
		return filepath.Join(homeDir, ".claude")
	}
	return workDir
}

func claudeCapabilities(frontmatter map[string]any) map[string]string {
	overrides := map[string]string{}
	readOnly := strings.EqualFold(exporter.StringValue(frontmatter, "permissionMode"), "plan") ||
		containsAny(exporter.StringSliceValue(frontmatter, "disallowedTools"), "Edit", "Write")
	if readOnly {
		overrides["edit_files"] = "deny"
		overrides["run_shell"] = "deny"
	} else {
		overrides["edit_files"] = "allow"
		overrides["run_shell"] = "ask"
	}

	tools := exporter.StringSliceValue(frontmatter, "tools")
	if containsAny(tools, "WebFetch") {
		overrides["fetch_urls"] = "allow"
	}
	if containsAny(tools, "WebSearch") {
		overrides["web_search"] = "allow"
	}
	if containsAny(tools, "Task") {
		overrides["spawn_agents"] = "allow"
	}
	return exporter.FullCapabilities(overrides)
}

func containsAny(values []string, targets ...string) bool {
	for _, value := range values {
		for _, target := range targets {
			if strings.EqualFold(strings.TrimSpace(value), target) {
				return true
			}
		}
	}
	return false
}

func defaultReasoning(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "medium"
	}
	return value
}
