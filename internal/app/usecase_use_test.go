package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
)

func TestUseCoordinatesPortsAndAppliesPlan(t *testing.T) {
	flow := testFlow()
	renderer := NewMockTargetRenderer(t)
	source := NewMockTemplateSource(t)
	loader := NewMockFlowLoader(t)
	collector := NewMockChoiceCollector(t)
	registry := NewMockTargetRegistry(t)
	planner := NewMockInstallPlanner(t)
	writer := NewMockInstallWriter(t)
	reporter := NewMockReporter(t)

	cleanupCalled := false
	resolved := ResolvedSource{
		Path: "template.yaml",
		Cleanup: func() {
			cleanupCalled = true
		},
	}
	choices := Choices{
		Target: binding.TargetCodex,
		Scope:  binding.ScopeProject,
		Models: binding.Models{"main": "gpt-test"},
	}
	artifacts := install.ArtifactSet{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Files: []install.DesiredFile{
			{Path: "AGENTS.md", Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	plan := install.Plan{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Actions: []install.Action{
			{Path: "AGENTS.md", Kind: install.ActionCreate, Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	renderInput := RenderInput{
		Flow:    flow,
		Models:  choices.Models,
		Scope:   choices.Scope,
		WorkDir: "/work",
		HomeDir: "/home",
	}

	reporter.On("Banner").Once()
	source.On("Resolve", mock.Anything, "repo", collector, reporter).Return(resolved, nil).Once()
	loader.On("LoadFile", "template.yaml").Return(LoadResult{Flow: flow}, nil).Once()
	renderer.On("Target").Return(binding.TargetCodex).Once()
	registry.On("All").Return([]TargetRenderer{renderer}).Once()
	collector.On("Collect", mock.Anything, flow, []TargetOption{{Value: binding.TargetCodex, Label: string(binding.TargetCodex)}}).Return(choices, nil).Once()
	registry.On("Get", string(binding.TargetCodex)).Return(renderer, nil).Once()
	renderer.On("Validate", mock.Anything, renderInput).Return([]diagnostic.Diagnostic(nil)).Once()
	renderer.On("Render", mock.Anything, renderInput).Return(artifacts, []diagnostic.Diagnostic(nil)).Once()
	planner.On("Build", artifacts).Return(plan).Once()
	allowHistoryBlock(reporter)
	collector.On("Confirm", mock.Anything, mock.Anything).Return(true, nil).Once()
	writer.On("Apply", plan).Return(nil).Once()
	reporter.On("Historyf", "Done.\n").Once()
	reporter.On("HistorySpace").Once()

	application := App{
		TemplateSource: source,
		FlowLoader:     loader,
		TargetRegistry: registry,
		InstallPlanner: planner,
		InstallWriter:  writer,
		Reporter:       reporter,
		WorkDir:        "/work",
		HomeDir:        "/home",
	}
	if err := application.Use(context.Background(), "repo", collector); err != nil {
		t.Fatal(err)
	}
	if !cleanupCalled {
		t.Fatal("source cleanup was not called")
	}
}

func TestUseStopsBeforeConfirmationWhenPlanHasConflicts(t *testing.T) {
	flow := testFlow()
	renderer := NewMockTargetRenderer(t)
	source := NewMockTemplateSource(t)
	loader := NewMockFlowLoader(t)
	collector := NewMockChoiceCollector(t)
	registry := NewMockTargetRegistry(t)
	planner := NewMockInstallPlanner(t)
	writer := NewMockInstallWriter(t)
	reporter := NewMockReporter(t)

	choices := Choices{
		Target: binding.TargetCodex,
		Scope:  binding.ScopeProject,
		Models: binding.Models{"main": "gpt-test"},
	}
	artifacts := install.ArtifactSet{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Files: []install.DesiredFile{
			{Path: "AGENTS.md", Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	plan := install.Plan{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Actions: []install.Action{
			{Path: "AGENTS.md", Kind: install.ActionConflict, Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	renderInput := RenderInput{
		Flow:    flow,
		Models:  choices.Models,
		Scope:   choices.Scope,
		WorkDir: "/work",
		HomeDir: "/home",
	}

	reporter.On("Banner").Once()
	source.On("Resolve", mock.Anything, "repo", collector, reporter).Return(ResolvedSource{Path: "template.yaml"}, nil).Once()
	loader.On("LoadFile", "template.yaml").Return(LoadResult{Flow: flow}, nil).Once()
	renderer.On("Target").Return(binding.TargetCodex).Once()
	registry.On("All").Return([]TargetRenderer{renderer}).Once()
	collector.On("Collect", mock.Anything, flow, []TargetOption{{Value: binding.TargetCodex, Label: string(binding.TargetCodex)}}).Return(choices, nil).Once()
	registry.On("Get", string(binding.TargetCodex)).Return(renderer, nil).Once()
	renderer.On("Validate", mock.Anything, renderInput).Return([]diagnostic.Diagnostic(nil)).Once()
	renderer.On("Render", mock.Anything, renderInput).Return(artifacts, []diagnostic.Diagnostic(nil)).Once()
	planner.On("Build", artifacts).Return(plan).Once()
	allowHistoryBlock(reporter)

	application := App{
		TemplateSource: source,
		FlowLoader:     loader,
		TargetRegistry: registry,
		InstallPlanner: planner,
		InstallWriter:  writer,
		Reporter:       reporter,
		WorkDir:        "/work",
		HomeDir:        "/home",
	}
	err := application.Use(context.Background(), "repo", collector)
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestUseDryRunPrintsPreviewWithoutConfirmingOrApplyingPlan(t *testing.T) {
	flow := testFlow()
	renderer := NewMockTargetRenderer(t)
	source := NewMockTemplateSource(t)
	loader := NewMockFlowLoader(t)
	collector := NewMockChoiceCollector(t)
	registry := NewMockTargetRegistry(t)
	planner := NewMockInstallPlanner(t)
	writer := NewMockInstallWriter(t)
	reporter := NewMockReporter(t)

	choices := Choices{
		Target: binding.TargetCodex,
		Scope:  binding.ScopeProject,
		Models: binding.Models{"main": "gpt-test"},
	}
	artifacts := install.ArtifactSet{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Files: []install.DesiredFile{
			{Path: "AGENTS.md", Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	plan := install.Plan{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Actions: []install.Action{
			{Path: "AGENTS.md", Kind: install.ActionCreate, Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	preview := install.FormatDryRunFilePreview(plan)
	renderInput := RenderInput{
		Flow:    flow,
		Models:  choices.Models,
		Scope:   choices.Scope,
		WorkDir: "/work",
		HomeDir: "/home",
	}

	reporter.On("Banner").Once()
	source.On("Resolve", mock.Anything, "repo", collector, reporter).Return(ResolvedSource{Path: "template.yaml"}, nil).Once()
	loader.On("LoadFile", "template.yaml").Return(LoadResult{Flow: flow}, nil).Once()
	renderer.On("Target").Return(binding.TargetCodex).Once()
	registry.On("All").Return([]TargetRenderer{renderer}).Once()
	collector.On("Collect", mock.Anything, flow, []TargetOption{{Value: binding.TargetCodex, Label: string(binding.TargetCodex)}}).Return(choices, nil).Once()
	registry.On("Get", string(binding.TargetCodex)).Return(renderer, nil).Once()
	renderer.On("Validate", mock.Anything, renderInput).Return([]diagnostic.Diagnostic(nil)).Once()
	renderer.On("Render", mock.Anything, renderInput).Return(artifacts, []diagnostic.Diagnostic(nil)).Once()
	planner.On("Build", artifacts).Return(plan).Once()
	allowHistoryBlock(reporter)
	reporter.On("MessageLine", []any{preview}).Once()

	application := App{
		TemplateSource: source,
		FlowLoader:     loader,
		TargetRegistry: registry,
		InstallPlanner: planner,
		InstallWriter:  writer,
		Reporter:       reporter,
		WorkDir:        "/work",
		HomeDir:        "/home",
	}
	if err := application.UseWithOptions(context.Background(), "repo", collector, UseOptions{DryRun: true}); err != nil {
		t.Fatal(err)
	}
}

func TestUseDryRunWithConflictsPrintsPreviewAndReturnsError(t *testing.T) {
	flow := testFlow()
	renderer := NewMockTargetRenderer(t)
	source := NewMockTemplateSource(t)
	loader := NewMockFlowLoader(t)
	collector := NewMockChoiceCollector(t)
	registry := NewMockTargetRegistry(t)
	planner := NewMockInstallPlanner(t)
	writer := NewMockInstallWriter(t)
	reporter := NewMockReporter(t)

	choices := Choices{
		Target: binding.TargetCodex,
		Scope:  binding.ScopeProject,
		Models: binding.Models{"main": "gpt-test"},
	}
	artifacts := install.ArtifactSet{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Files: []install.DesiredFile{
			{Path: "AGENTS.md", Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	plan := install.Plan{
		Target: string(binding.TargetCodex),
		Scope:  string(binding.ScopeProject),
		Actions: []install.Action{
			{Path: "AGENTS.md", Kind: install.ActionConflict, Content: []byte("# Test"), Strategy: install.StrategyCreateOnly},
		},
	}
	preview := install.FormatDryRunFilePreview(plan)
	renderInput := RenderInput{
		Flow:    flow,
		Models:  choices.Models,
		Scope:   choices.Scope,
		WorkDir: "/work",
		HomeDir: "/home",
	}

	reporter.On("Banner").Once()
	source.On("Resolve", mock.Anything, "repo", collector, reporter).Return(ResolvedSource{Path: "template.yaml"}, nil).Once()
	loader.On("LoadFile", "template.yaml").Return(LoadResult{Flow: flow}, nil).Once()
	renderer.On("Target").Return(binding.TargetCodex).Once()
	registry.On("All").Return([]TargetRenderer{renderer}).Once()
	collector.On("Collect", mock.Anything, flow, []TargetOption{{Value: binding.TargetCodex, Label: string(binding.TargetCodex)}}).Return(choices, nil).Once()
	registry.On("Get", string(binding.TargetCodex)).Return(renderer, nil).Once()
	renderer.On("Validate", mock.Anything, renderInput).Return([]diagnostic.Diagnostic(nil)).Once()
	renderer.On("Render", mock.Anything, renderInput).Return(artifacts, []diagnostic.Diagnostic(nil)).Once()
	planner.On("Build", artifacts).Return(plan).Once()
	allowHistoryBlock(reporter)
	reporter.On("MessageLine", []any{preview}).Once()

	application := App{
		TemplateSource: source,
		FlowLoader:     loader,
		TargetRegistry: registry,
		InstallPlanner: planner,
		InstallWriter:  writer,
		Reporter:       reporter,
		WorkDir:        "/work",
		HomeDir:        "/home",
	}
	err := application.UseWithOptions(context.Background(), "repo", collector, UseOptions{DryRun: true})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func allowHistoryBlock(reporter *MockReporter) {
	reporter.On("HistoryBlock", mock.Anything).Maybe()
}

func testFlow() flowmodel.Flow {
	return flowmodel.Flow{
		ID:      "test-flow",
		Version: 1,
		ModelSlots: map[string]flowmodel.ModelSlot{
			"main": {},
		},
		PermissionProfiles: map[string]flowmodel.PermissionProfile{
			"read": {
				Capabilities: map[string]string{
					"edit_files": "deny",
				},
			},
		},
		Agents: map[string]flowmodel.Agent{
			"reviewer": {
				ID:                "reviewer",
				Description:       "Reviews code",
				ModelSlot:         "main",
				ReasoningEffort:   "medium",
				PermissionProfile: "read",
				Prompt:            "Review code.",
			},
		},
		Instructions: map[string]string{"AGENTS.md": "# Test"},
	}
}
