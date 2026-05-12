package install

import (
	"fmt"
	"os"
	"path/filepath"
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
		case ActionCreate:
			if err := writeAtomic(action.Path, action.Content); err != nil {
				return err
			}
		case ActionUpdate:
			if err := writeAtomic(action.Path, action.Content); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported action %q for %s", action.Kind, action.Path)
		}
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".agentflow-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %q: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	return nil
}
