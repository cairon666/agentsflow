package source

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cairon666/agentsflow/internal/console"
)

func TestResolveReturnsLocalTemplatePath(t *testing.T) {
	resolver := NewResolver()

	path, cleanup, err := resolver.Resolve(t.Context(), " template.yaml ", nil, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if path != "template.yaml" {
		t.Fatalf("path = %q, want template.yaml", path)
	}
}

func TestResolveRemoteRepositoryPromptsForSingleTemplate(t *testing.T) {
	repoDir := t.TempDir()
	writeRemoteTemplate(t, repoDir, "alpha", "template")

	cloner := &fakeGitCloner{sourceDir: repoDir}
	var stdout bytes.Buffer
	chooser := &recordingTemplateChooser{selectedLabel: "alpha"}
	resolver := DefaultResolver{Cloner: cloner}

	path, cleanup, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", chooser, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup == nil {
		t.Fatal("cleanup is nil")
	}
	defer cleanup()

	assertSelectedTemplate(t, path, cloner.dest, "alpha")
	if chooser.templateCalls != 1 {
		t.Fatalf("template prompt calls = %d, want 1", chooser.templateCalls)
	}
	if !reflect.DeepEqual(chooser.labels, []string{"alpha"}) {
		t.Fatalf("template labels = %v, want [alpha]", chooser.labels)
	}
	cleanup()
	assertTempRepoRemoved(t, cloner.dest)
}

func TestResolveRemoteRepositorySortsAndUsesSelectedTemplate(t *testing.T) {
	repoDir := t.TempDir()
	writeRemoteTemplate(t, repoDir, "beta", "beta")
	writeRemoteTemplate(t, repoDir, "alpha", "alpha")

	chooser := &recordingTemplateChooser{selectedLabel: "beta"}
	cloner := &fakeGitCloner{sourceDir: repoDir}
	resolver := DefaultResolver{Cloner: cloner}

	path, cleanup, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", chooser, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if !reflect.DeepEqual(chooser.labels, []string{"alpha", "beta"}) {
		t.Fatalf("template labels = %v, want [alpha beta]", chooser.labels)
	}
	assertSelectedTemplate(t, path, cloner.dest, "beta")
	cleanup()
	assertTempRepoRemoved(t, cloner.dest)
}

func TestResolveRemoteRepositoryRequiresTemplates(t *testing.T) {
	repoDir := t.TempDir()

	cloner := &fakeGitCloner{sourceDir: repoDir}
	chooser := &recordingTemplateChooser{selectedLabel: "alpha"}
	resolver := DefaultResolver{Cloner: cloner}

	_, _, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", chooser, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	assertTempRepoRemoved(t, cloner.dest)
	if !strings.Contains(err.Error(), "no templates found") {
		t.Fatalf("error = %q, want no templates found", err)
	}
	if chooser.templateCalls != 0 {
		t.Fatalf("template prompt calls = %d, want 0", chooser.templateCalls)
	}
}

func TestResolveRemoteRepositoryRemovesTempDirWhenCloneFails(t *testing.T) {
	cloneErr := errors.New("network down")
	cloner := &failingGitCloner{err: cloneErr}
	resolver := DefaultResolver{Cloner: cloner}

	_, _, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", &recordingTemplateChooser{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cloneErr) {
		t.Fatalf("error = %v, want wrapped %v", err, cloneErr)
	}
	assertTempRepoRemoved(t, cloner.dest)
}

func TestResolveRemoteRepositoryCancelsClone(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cloner := &cancelingGitCloner{cancel: cancel}
	resolver := DefaultResolver{Cloner: cloner}

	_, _, err := resolver.Resolve(ctx, "https://example.test/repo.git", &recordingTemplateChooser{}, &bytes.Buffer{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	assertTempRepoRemoved(t, cloner.dest)
}

func TestResolveRemoteRepositoryWaitsForCloneAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cloner := newBlockingCancelGitCloner()
	resolver := DefaultResolver{Cloner: cloner}
	errCh := make(chan error, 1)

	go func() {
		_, _, err := resolver.Resolve(ctx, "https://example.test/repo.git", &recordingTemplateChooser{}, &bytes.Buffer{})
		errCh <- err
	}()

	<-cloner.started
	cancel()
	<-cloner.cancelled
	select {
	case err := <-errCh:
		t.Fatalf("Resolve returned before clone completed: %v", err)
	default:
	}

	cloner.release()
	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	assertTempRepoRemoved(t, cloner.dest)
}

func TestRunWithLoadingUsesAccessibleModeForNonTerminalOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := console.RunWithLoading(t.Context(), &stdout, "Loading repository...", func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Loading repository") {
		t.Fatalf("stdout missing loading title:\n%s", output)
	}
	if strings.Contains(output, "[?2026") || strings.Contains(output, "[?2027") || strings.Contains(output, "]11;") {
		t.Fatalf("stdout included terminal query sequences:\n%q", output)
	}
}

func assertSelectedTemplate(t *testing.T, path, repoDest, name string) {
	t.Helper()
	want := filepath.Join(repoDest, templateRepoDir, name, "template.yaml")
	if path != want {
		t.Fatalf("selected template = %q, want %q", path, want)
	}
}

func assertTempRepoRemoved(t *testing.T, repoDest string) {
	t.Helper()
	if repoDest == "" {
		t.Fatal("git cloner did not receive a destination")
	}
	root := filepath.Dir(repoDest)
	if !strings.HasPrefix(filepath.Base(root), "agentsflow-") {
		t.Fatalf("temporary repository root = %q, want prefix agentsflow-", root)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary repository root still exists or could not be inspected: %v", err)
	}
}

func writeRemoteTemplate(t *testing.T, repoDir, name, content string) {
	t.Helper()
	path := filepath.Join(repoDir, templateRepoDir, name, "template.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type recordingTemplateChooser struct {
	selectedLabel string
	templateCalls int
	labels        []string
}

func (c *recordingTemplateChooser) ChooseTemplate(options []TemplateOption) (string, error) {
	c.templateCalls++
	c.labels = c.labels[:0]
	for _, option := range options {
		c.labels = append(c.labels, option.Label)
		if option.Label == c.selectedLabel {
			return option.Value, nil
		}
	}
	return options[0].Value, nil
}

type fakeGitCloner struct {
	sourceDir string
	dest      string
}

func (c *fakeGitCloner) Clone(_ context.Context, _, dest string) error {
	c.dest = dest
	return copyTree(c.sourceDir, dest)
}

type failingGitCloner struct {
	dest string
	err  error
}

func (c *failingGitCloner) Clone(_ context.Context, _, dest string) error {
	c.dest = dest
	return c.err
}

type cancelingGitCloner struct {
	dest   string
	cancel context.CancelFunc
}

func (c *cancelingGitCloner) Clone(ctx context.Context, _, dest string) error {
	c.dest = dest
	if c.cancel != nil {
		c.cancel()
	}
	<-ctx.Done()
	return ctx.Err()
}

type blockingCancelGitCloner struct {
	dest      string
	started   chan struct{}
	cancelled chan struct{}
	released  chan struct{}
}

func newBlockingCancelGitCloner() *blockingCancelGitCloner {
	return &blockingCancelGitCloner{
		started:   make(chan struct{}),
		cancelled: make(chan struct{}),
		released:  make(chan struct{}),
	}
}

func (c *blockingCancelGitCloner) Clone(ctx context.Context, _, dest string) error {
	c.dest = dest
	close(c.started)
	<-ctx.Done()
	close(c.cancelled)
	<-c.released
	return ctx.Err()
}

func (c *blockingCancelGitCloner) release() {
	close(c.released)
}

func copyTree(source, dest string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
