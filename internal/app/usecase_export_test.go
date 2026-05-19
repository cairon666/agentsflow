package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/cairon666/agentsflow/internal/binding"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
)

func TestExportCoordinatesPortsAndAppliesPlan(t *testing.T) {
	spec := testExportSpec()
	sourceExporter := NewMockSourceExporter(t)
	collector := NewMockExportChoiceCollector(t)
	registry := NewMockExporterRegistry(t)
	encoder := NewMockSpecEncoder(t)
	planner := NewMockInstallPlanner(t)
	writer := NewMockInstallWriter(t)
	reporter := NewMockReporter(t)

	choices := ExportChoices{
		Source: binding.TargetCodex,
		Scope:  binding.ScopeProject,
		Output: "agentsflow.yaml",
	}
	exportInput := ExportInput{
		Source:  binding.TargetCodex,
		Scope:   binding.ScopeProject,
		WorkDir: "/work",
		HomeDir: "/home",
	}
	content := []byte("id: exported-codex-project\n")
	artifacts := install.ArtifactSet{
		Target: "export",
		Scope:  string(binding.ScopeProject),
		Files: []install.DesiredFile{
			{Path: "/work/agentsflow.yaml", Content: content, Strategy: install.StrategyOverwrite},
		},
	}
	plan := install.Plan{
		Target: "export",
		Scope:  string(binding.ScopeProject),
		Actions: []install.Action{
			{Path: "agentsflow.yaml", Kind: install.ActionCreate, Content: content, Strategy: install.StrategyOverwrite},
		},
	}

	reporter.On("Banner").Once()
	sourceExporter.On("Source").Return(binding.TargetCodex).Once()
	registry.On("All").Return([]SourceExporter{sourceExporter}).Once()
	collector.On("CollectExport", mock.Anything, []ExportSourceOption{{Value: binding.TargetCodex, Label: string(binding.TargetCodex)}}).Return(choices, nil).Once()
	registry.On("Get", string(binding.TargetCodex)).Return(sourceExporter, nil).Once()
	sourceExporter.On("Export", mock.Anything, exportInput).Return(ExportResult{Spec: spec}, nil).Once()
	encoder.On("EncodeSpec", spec).Return(content, nil).Once()
	planner.On("Build", artifacts).Return(plan).Once()
	allowHistoryBlock(reporter)
	collector.On("Confirm", mock.Anything, mock.Anything).Return(true, nil).Once()
	writer.On("Apply", plan).Return(nil).Once()
	reporter.On("Historyf", "Done.\n").Once()
	reporter.On("HistorySpace").Once()

	application := App{
		ExporterRegistry: registry,
		SpecEncoder:      encoder,
		InstallPlanner:   planner,
		InstallWriter:    writer,
		Reporter:         reporter,
		WorkDir:          "/work",
		HomeDir:          "/home",
	}
	if err := application.Export(context.Background(), collector); err != nil {
		t.Fatal(err)
	}
}

func testExportSpec() flowmodel.Spec {
	return flowmodel.Spec{
		ID:      "exported-codex-project",
		Version: 1,
		ModelSlots: map[string]flowmodel.SpecModelSlot{
			"main": {
				Description: "Main model exported from Codex.",
			},
		},
		PermissionProfiles: map[string]flowmodel.SpecPermissionProfile{
			"read_only": {
				Description: "Read only.",
				Capabilities: map[string]string{
					"read_files": "allow",
					"edit_files": "deny",
				},
			},
		},
		Agents: map[string]flowmodel.SpecAgent{
			"reviewer": {
				Description:       "Reviews code.",
				ModelSlot:         "main",
				ReasoningEffort:   "high",
				PermissionProfile: "read_only",
				Prompt:            "Review.",
			},
		},
		Instructions: map[string]string{
			"AGENTS.md": "# Shared",
		},
	}
}
