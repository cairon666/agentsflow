package app

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/x/term"

	"github.com/cairon666/agentsflow/internal/builder"
	"github.com/cairon666/agentsflow/internal/console"
)

const templateRepoDir = ".agentsflow"

func (a App) resolveTemplateSource(ctx context.Context, source string, prompter builder.Prompter) (string, func(), error) {
	source = strings.TrimSpace(source)
	if !isGitSource(source) {
		return source, func() {}, nil
	}
	return a.resolveGitTemplate(ctx, source, prompter)
}

func (a App) resolveGitTemplate(ctx context.Context, source string, prompter builder.Prompter) (string, func(), error) {
	chooser, ok := prompter.(builder.TemplatePrompter)
	if !ok {
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
	cloner := a.GitCloner
	if cloner == nil {
		cloner = gitCLICloner{}
	}
	if err := runWithLoading(ctx, a.Stdout, "Loading repository...", func(ctx context.Context) error {
		return cloner.Clone(ctx, source, repoDir)
	}); err != nil {
		cleanup()
		return "", nil, err
	}

	console.NewHistoryWriter(a.Stdout).WriteHistorySpace().WriteHistoryf("Source: %s\n", source)

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

func discoverTemplateOptions(repoDir string) ([]builder.TemplateOption, error) {
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

	options := make([]builder.TemplateOption, 0, len(matches))
	for _, match := range matches {
		name := templateName(match)
		options = append(options, builder.TemplateOption{
			Value: match,
			Label: name,
		})
	}
	return options, nil
}

func templateName(path string) string {
	return filepath.Base(filepath.Dir(path))
}

func isGitSource(value string) bool {
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

type gitCLICloner struct{}

func (gitCLICloner) Clone(ctx context.Context, source, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", source, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("clone template repository: %w: %s", err, message)
		}
		return fmt.Errorf("clone template repository: %w", err)
	}
	return nil
}

func runWithLoading(ctx context.Context, out io.Writer, title string, action func(context.Context) error) error {
	if out == nil {
		out = io.Discard
	}
	loading := spinner.New().
		Title(title).
		WithTheme(spinner.ThemeFunc(func(bool) *spinner.Styles {
			return spinner.ThemeDefault(true)
		})).
		WithViewHook(func(v tea.View) tea.View {
			v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, 1)
			return v
		}).
		WithOutput(out).
		ActionWithErr(func(context.Context) error {
			return action(ctx)
		})
	if !isTerminalWriter(out) {
		loading = loading.WithAccessible(true)
	}
	return loading.Run()
}

func isTerminalWriter(out io.Writer) bool {
	file, ok := out.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(file.Fd())
}
