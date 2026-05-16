package source

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestResolveReturnsLocalTemplatePath(t *testing.T) {
	resolver := NewResolver()
	reporter := NewMockReporter(t)

	path, cleanup, err := resolver.Resolve(t.Context(), " template.yaml ", nil, reporter)
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

	var cloneDest string
	cloner := NewMockCloner(t)
	expectCloneCopies(cloner, "https://example.test/repo.git", repoDir, &cloneDest)
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	expectSourceHistory(reporter, "https://example.test/repo.git")
	chooser := &recordingTemplateChooser{selectedLabel: "alpha"}
	resolver := DefaultResolver{Cloner: cloner}

	path, cleanup, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", chooser, reporter)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup == nil {
		t.Fatal("cleanup is nil")
	}
	defer cleanup()

	assertSelectedTemplate(t, path, cloneDest, "alpha")
	if chooser.templateCalls != 1 {
		t.Fatalf("template prompt calls = %d, want 1", chooser.templateCalls)
	}
	if !reflect.DeepEqual(chooser.labels, []string{"alpha"}) {
		t.Fatalf("template labels = %v, want [alpha]", chooser.labels)
	}
	cleanup()
	assertTempRepoRemoved(t, cloneDest)
}

func TestResolveRemoteRepositorySortsAndUsesSelectedTemplate(t *testing.T) {
	repoDir := t.TempDir()
	writeRemoteTemplate(t, repoDir, "beta", "beta")
	writeRemoteTemplate(t, repoDir, "alpha", "alpha")

	chooser := &recordingTemplateChooser{selectedLabel: "beta"}
	var cloneDest string
	cloner := NewMockCloner(t)
	expectCloneCopies(cloner, "https://example.test/repo.git", repoDir, &cloneDest)
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	expectSourceHistory(reporter, "https://example.test/repo.git")
	resolver := DefaultResolver{Cloner: cloner}

	path, cleanup, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", chooser, reporter)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if !reflect.DeepEqual(chooser.labels, []string{"alpha", "beta"}) {
		t.Fatalf("template labels = %v, want [alpha beta]", chooser.labels)
	}
	assertSelectedTemplate(t, path, cloneDest, "beta")
	cleanup()
	assertTempRepoRemoved(t, cloneDest)
}

func TestResolveRemoteRepositoryRequiresTemplates(t *testing.T) {
	repoDir := t.TempDir()

	var cloneDest string
	cloner := NewMockCloner(t)
	expectCloneCopies(cloner, "https://example.test/repo.git", repoDir, &cloneDest)
	chooser := &recordingTemplateChooser{selectedLabel: "alpha"}
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	expectSourceHistory(reporter, "https://example.test/repo.git")
	resolver := DefaultResolver{Cloner: cloner}

	_, _, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", chooser, reporter)
	if err == nil {
		t.Fatal("expected error")
	}
	assertTempRepoRemoved(t, cloneDest)
	if !strings.Contains(err.Error(), "no templates found") {
		t.Fatalf("error = %q, want no templates found", err)
	}
	if chooser.templateCalls != 0 {
		t.Fatalf("template prompt calls = %d, want 0", chooser.templateCalls)
	}
}

func TestResolveRemoteRepositoryRemovesTempDirWhenHistoryPanics(t *testing.T) {
	writeErr := errors.New("write failed")
	var cloneDest string
	cloner := NewMockCloner(t)
	expectCloneCopies(cloner, "https://example.test/repo.git", t.TempDir(), &cloneDest)
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	reporter.On("HistorySpace").Run(func(mock.Arguments) {
		panic(writeErr)
	}).Once()
	resolver := DefaultResolver{Cloner: cloner}

	recovered := func() (recovered any) {
		defer func() {
			recovered = recover()
		}()

		_, _, _ = resolver.Resolve(t.Context(), "https://example.test/repo.git", &recordingTemplateChooser{}, reporter)
		return nil
	}()
	if recovered == nil {
		t.Fatal("expected panic")
	}
	err, ok := recovered.(error)
	if !ok {
		t.Fatalf("panic = %T %v, want error", recovered, recovered)
	}
	if !errors.Is(err, writeErr) {
		t.Fatalf("panic = %v, want wrapped %v", err, writeErr)
	}
	assertTempRepoRemoved(t, cloneDest)
}

func TestResolveRemoteRepositoryRemovesTempDirWhenCloneFails(t *testing.T) {
	cloneErr := errors.New("network down")
	var cloneDest string
	cloner := NewMockCloner(t)
	expectClone(cloner, "https://example.test/repo.git", &cloneDest, func(context.Context, string) error {
		return cloneErr
	})
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	resolver := DefaultResolver{Cloner: cloner}

	_, _, err := resolver.Resolve(t.Context(), "https://example.test/repo.git", &recordingTemplateChooser{}, reporter)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cloneErr) {
		t.Fatalf("error = %v, want wrapped %v", err, cloneErr)
	}
	assertTempRepoRemoved(t, cloneDest)
}

func TestResolveRemoteRepositoryCancelsClone(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var cloneDest string
	cloner := NewMockCloner(t)
	expectClone(cloner, "https://example.test/repo.git", &cloneDest, func(ctx context.Context, _ string) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	})
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	resolver := DefaultResolver{Cloner: cloner}

	_, _, err := resolver.Resolve(ctx, "https://example.test/repo.git", &recordingTemplateChooser{}, reporter)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	assertTempRepoRemoved(t, cloneDest)
}

func TestResolveRemoteRepositoryWaitsForCloneAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var cloneDest string
	started := make(chan struct{})
	cancelled := make(chan struct{})
	released := make(chan struct{})
	cloner := NewMockCloner(t)
	expectClone(cloner, "https://example.test/repo.git", &cloneDest, func(ctx context.Context, _ string) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		<-released
		return ctx.Err()
	})
	reporter := NewMockReporter(t)
	expectLoadingRuns(reporter)
	resolver := DefaultResolver{Cloner: cloner}
	errCh := make(chan error, 1)

	go func() {
		_, _, err := resolver.Resolve(ctx, "https://example.test/repo.git", &recordingTemplateChooser{}, reporter)
		errCh <- err
	}()

	<-started
	cancel()
	<-cancelled
	select {
	case err := <-errCh:
		t.Fatalf("Resolve returned before clone completed: %v", err)
	default:
	}

	close(released)
	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	assertTempRepoRemoved(t, cloneDest)
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

func expectLoadingRuns(reporter *MockReporter) {
	reporter.On("RunLoading", mock.Anything, "Loading repository...", mock.Anything).
		Return(func(ctx context.Context, _ string, action func(context.Context) error) error {
			return action(ctx)
		}).
		Once()
}

func expectSourceHistory(reporter *MockReporter, source string) {
	reporter.On("HistorySpace").Once()
	reporter.On("Historyf", "Source: %s\n", []any{source}).Once()
}

func expectCloneCopies(cloner *MockCloner, source, sourceDir string, cloneDest *string) {
	expectClone(cloner, source, cloneDest, func(_ context.Context, dest string) error {
		return copyTree(sourceDir, dest)
	})
}

func expectClone(cloner *MockCloner, source string, cloneDest *string, clone func(context.Context, string) error) {
	cloner.On("Clone", mock.Anything, source, mock.Anything).
		Return(func(ctx context.Context, _ string, dest string) error {
			*cloneDest = dest
			return clone(ctx, dest)
		}).
		Once()
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
