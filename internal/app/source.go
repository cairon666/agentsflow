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
	"time"

	"github.com/cairon666/agentsflow/internal/builder"
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
	if err := runWithLoading(a.Stdout, "Loading repository", func() error {
		return cloner.Clone(ctx, source, repoDir)
	}); err != nil {
		cleanup()
		return "", nil, err
	}

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

func runWithLoading(out io.Writer, title string, action func() error) error {
	if out == nil {
		out = io.Discard
	}
	done := make(chan error, 1)
	go func() {
		done <- action()
	}()

	frames := []string{
		"[>         ]",
		"[=>        ]",
		"[==>       ]",
		"[===>      ]",
		"[====>     ]",
		"[=====>    ]",
		"[======>   ]",
		"[=======>  ]",
		"[========> ]",
		"[=========>]",
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	index := 0
	fmt.Fprintf(out, "\r%s %s", title, frames[index])
	for {
		select {
		case err := <-done:
			if err != nil {
				fmt.Fprintf(out, "\r%s failed\n", title)
				return err
			}
			fmt.Fprintf(out, "\r%s done\n", title)
			return nil
		case <-ticker.C:
			index = (index + 1) % len(frames)
			fmt.Fprintf(out, "\r%s %s", title, frames[index])
		}
	}
}
