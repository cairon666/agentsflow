package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Writer applies install plans to disk.
type Writer struct{}

// NewWriter creates a filesystem writer.
func NewWriter() Writer {
	return Writer{}
}

// Apply writes all create/update actions from plan.
func (w Writer) Apply(plan Plan) error {
	if plan.HasConflicts() {
		return fmt.Errorf("install plan has conflicts")
	}
	for _, action := range plan.Actions {
		switch action.Kind {
		case ActionSkip:
			continue
		case ActionCleanDir:
			if err := cleanDirContents(action.Path); err != nil {
				return err
			}
		case ActionCreate:
			if err := writeAtomic(action.Path, action.Content); err != nil {
				return err
			}
		case ActionUpdate:
			if err := writeAtomic(action.Path, action.Content); err != nil {
				return err
			}
		case ActionOverwrite:
			if err := writeAtomic(action.Path, action.Content); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported action %q for %s", action.Kind, action.Path)
		}
	}
	return nil
}

func cleanDirContents(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("clean directory path is empty")
	}
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == ".." ||
		cleanPath == string(filepath.Separator) ||
		strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to clean unsafe directory %q", path)
	}
	info, err := os.Lstat(cleanPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect directory %q: %w", cleanPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to clean symlink directory %q", cleanPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to clean non-directory %q", cleanPath)
	}
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return fmt.Errorf("read directory %q: %w", cleanPath, err)
	}
	for _, entry := range entries {
		entryPath := filepath.Join(cleanPath, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("remove %q: %w", entryPath, err)
		}
	}
	return nil
}

func writeAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".agentsflow-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", path, err)
	}
	tmpName := tmp.Name()
	shouldRemove := true
	defer func() {
		if shouldRemove {
			err = errors.Join(err, os.Remove(tmpName))
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("write temp file for %q: %w", path, err),
				fmt.Errorf("close temp file for %q: %w", path, closeErr),
			)
		}
		return fmt.Errorf("write temp file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %q: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	shouldRemove = false
	return nil
}
