package composition

import (
	"context"
	"io"
	"os"

	"github.com/cairon666/agentsflow/internal/app"
	"github.com/cairon666/agentsflow/internal/binding"
	"github.com/cairon666/agentsflow/internal/console"
	"github.com/cairon666/agentsflow/internal/diagnostic"
	flowmodel "github.com/cairon666/agentsflow/internal/flow"
	"github.com/cairon666/agentsflow/internal/install"
	templatesource "github.com/cairon666/agentsflow/internal/source"
	"github.com/cairon666/agentsflow/internal/target"
	"github.com/cairon666/agentsflow/internal/target/claude"
	"github.com/cairon666/agentsflow/internal/target/codex"
	"github.com/cairon666/agentsflow/internal/target/opencode"
)

// Config controls application composition for the CLI binary.
type Config struct {
	Stdout io.Writer
}

// NewApp creates the application use case with production dependencies.
func NewApp(config Config) app.App {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return app.App{
		TemplateSource: sourceResolverPort{resolver: templatesource.NewResolver()},
		FlowLoader:     flowLoaderPort{},
		TargetRegistry: targetRegistryPort{registry: target.NewRegistry(
			codex.New(),
			claude.New(),
			opencode.New(),
		)},
		InstallPlanner: installPlannerPort{},
		InstallWriter:  install.NewWriter(),
		Reporter:       console.NewReporter(config.Stdout),
		WorkDir:        workDir,
		HomeDir:        homeDir,
	}
}

type sourceResolverPort struct {
	resolver templatesource.Resolver
}

func (p sourceResolverPort) Resolve(ctx context.Context, source string, chooser app.TemplateChooser, reporter app.Reporter) (app.ResolvedSource, error) {
	path, cleanup, err := p.resolver.Resolve(ctx, source, sourceTemplateChooser{chooser: chooser}, reporter)
	if err != nil {
		return app.ResolvedSource{}, err
	}
	return app.ResolvedSource{Path: path, Cleanup: cleanup}, nil
}

type sourceTemplateChooser struct {
	chooser app.TemplateChooser
}

func (c sourceTemplateChooser) ChooseTemplate(options []templatesource.TemplateOption) (string, error) {
	appOptions := make([]app.TemplateOption, 0, len(options))
	for _, option := range options {
		appOptions = append(appOptions, app.TemplateOption{
			Value: option.Value,
			Label: option.Label,
		})
	}
	return c.chooser.ChooseTemplate(appOptions)
}

type flowLoaderPort struct{}

func (flowLoaderPort) LoadFile(path string) (app.LoadResult, error) {
	loaded, err := flowmodel.LoadFile(path)
	if err != nil {
		return app.LoadResult{}, err
	}
	return app.LoadResult{Flow: loaded.Flow, Diagnostics: loaded.Diagnostics}, nil
}

type targetRegistryPort struct {
	registry target.Registry
}

func (r targetRegistryPort) Resolve(name string) (binding.Target, error) {
	return r.registry.Resolve(name)
}

func (r targetRegistryPort) Get(name string) (app.TargetRenderer, error) {
	renderer, err := r.registry.Get(name)
	if err != nil {
		return nil, err
	}
	return targetRendererPort{renderer: renderer}, nil
}

func (r targetRegistryPort) All() []app.TargetRenderer {
	renderers := r.registry.All()
	out := make([]app.TargetRenderer, 0, len(renderers))
	for _, renderer := range renderers {
		out = append(out, targetRendererPort{renderer: renderer})
	}
	return out
}

type targetRendererPort struct {
	renderer target.Renderer
}

func (r targetRendererPort) Target() binding.Target {
	return r.renderer.Metadata().Name
}

func (r targetRendererPort) Validate(ctx context.Context, input app.RenderInput) []diagnostic.Diagnostic {
	return r.renderer.Validate(ctx, target.RenderInput{
		Flow:    input.Flow,
		Models:  input.Models,
		Scope:   input.Scope,
		WorkDir: input.WorkDir,
		HomeDir: input.HomeDir,
	})
}

func (r targetRendererPort) Render(ctx context.Context, input app.RenderInput) (install.ArtifactSet, []diagnostic.Diagnostic) {
	return r.renderer.Render(ctx, target.RenderInput{
		Flow:    input.Flow,
		Models:  input.Models,
		Scope:   input.Scope,
		WorkDir: input.WorkDir,
		HomeDir: input.HomeDir,
	})
}

type installPlannerPort struct{}

func (installPlannerPort) Build(artifacts install.ArtifactSet) install.Plan {
	return install.BuildPlan(artifacts)
}
