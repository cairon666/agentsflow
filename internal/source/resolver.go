package source

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cairon666/agentsflow/internal/console"
)

const templateRepoDir = ".agentsflow"

// TemplateOption is shown to the user when a repository contains templates.
type TemplateOption struct {
	Value string
	Label string
}

// TemplateChooser chooses a template from a repository source.
type TemplateChooser interface {
	ChooseTemplate([]TemplateOption) (string, error)
}

// Cloner clones a git repository into a destination directory.
type Cloner interface {
	Clone(context.Context, string, string) error
}

// LoadingRunner runs a source-loading action with optional progress output.
type LoadingRunner interface {
	Run(context.Context, io.Writer, string, func(context.Context) error) error
}

// LoadingRunnerFunc adapts a function to LoadingRunner.
type LoadingRunnerFunc func(context.Context, io.Writer, string, func(context.Context) error) error

// Run runs f.
func (f LoadingRunnerFunc) Run(ctx context.Context, out io.Writer, title string, action func(context.Context) error) error {
	return f(ctx, out, title, action)
}

// Resolver resolves a template source to a local template file path.
type Resolver interface {
	Resolve(context.Context, string, TemplateChooser, io.Writer) (string, func(), error)
}

// DefaultResolver resolves local paths and git repository sources.
type DefaultResolver struct {
	Cloner  Cloner
	Loading LoadingRunner
}

// NewResolver creates a resolver.
func NewResolver() DefaultResolver {
	return DefaultResolver{}
}

// Resolve resolves a local template path or a git repository source.
func (r DefaultResolver) Resolve(ctx context.Context, source string, chooser TemplateChooser, out io.Writer) (string, func(), error) {
	source = strings.TrimSpace(source)
	if !IsGitSource(source) {
		return source, func() {}, nil
	}
	return r.resolveGitTemplate(ctx, source, chooser, out)
}

func (r DefaultResolver) resolveGitTemplate(ctx context.Context, source string, chooser TemplateChooser, out io.Writer) (string, func(), error) {
	if chooser == nil {
		return "", nil, fmt.Errorf("template selection prompt unavailable")
	}

	root, err := os.MkdirTemp("", "agentsflow-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary repository directory: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(root)
	}

	repoDir := filepath.Join(root, "repo")
	cloner := r.Cloner
	if cloner == nil {
		cloner = GitCLICloner{}
	}
	loading := r.Loading
	if loading == nil {
		loading = LoadingRunnerFunc(console.RunWithLoading)
	}
	if err := loading.Run(ctx, out, "Loading repository...", func(ctx context.Context) error {
		return cloner.Clone(ctx, source, repoDir)
	}); err != nil {
		cleanup()
		return "", nil, err
	}

	console.NewHistoryWriter(out).WriteHistorySpace().WriteHistoryf("Source: %s\n", source)

	options, err := discoverTemplateOptions(repoDir)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	selected, err := chooser.ChooseTemplate(options)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("choose template: %w", err)
	}
	if selected == "" {
		cleanup()
		return "", nil, fmt.Errorf("choose template: selected template is empty")
	}
	return selected, cleanup, nil
}

func discoverTemplateOptions(repoDir string) ([]TemplateOption, error) {
	pattern := filepath.Join(repoDir, templateRepoDir, "*", "template.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("find templates: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no templates found; expected %s/<name>/template.yaml", templateRepoDir)
	}
	sort.Slice(matches, func(i, j int) bool {
		return templateName(matches[i]) < templateName(matches[j])
	})

	options := make([]TemplateOption, 0, len(matches))
	for _, match := range matches {
		name := templateName(match)
		options = append(options, TemplateOption{
			Value: match,
			Label: name,
		})
	}
	return options, nil
}

func templateName(path string) string {
	return filepath.Base(filepath.Dir(path))
}

// IsGitSource reports whether value points to a git repository source.
func IsGitSource(value string) bool {
	if strings.HasPrefix(value, "git@") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "git", "http", "https", "ssh", "file":
		return parsed.Host != "" || parsed.Scheme == "file"
	default:
		return false
	}
}
