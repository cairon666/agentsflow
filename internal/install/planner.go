package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildPlan resolves rendered artifacts into create/update/skip/conflict actions.
func BuildPlan(artifacts ArtifactSet) Plan {
	cleanDirs := normalizedCleanDirs(artifacts.CleanDirs)
	actions := make([]Action, 0, len(cleanDirs)+len(artifacts.Files))
	for _, dir := range cleanDirs {
		actions = append(actions, Action{Path: dir, Kind: ActionCleanDir})
	}
	for _, file := range artifacts.Files {
		actions = append(actions, classifyAction(file.Path, file.Content, normalizedStrategy(file.Strategy), cleanDirs))
	}
	return Plan{Target: artifacts.Target, Scope: artifacts.Scope, Actions: actions}
}

func classifyAction(path string, content []byte, strategy FileStrategy, cleanDirs []string) Action {
	action := Action{Path: path, Content: content, Strategy: strategy}
	existing, err := os.ReadFile(path)
	if isInsideAnyCleanDir(path, cleanDirs) {
		action.Kind = ActionCreate
		return action
	}
	if err == nil {
		action.ExistingContent = existing
	}
	switch {
	case err == nil && bytes.Equal(existing, content):
		action.Kind = ActionSkip
	case err == nil && strategy == StrategyMerge:
		action.Kind = ActionUpdate
	case err == nil && strategy == StrategyOverwrite:
		action.Kind = ActionOverwrite
	case err == nil:
		action.Kind = ActionConflict
	case os.IsNotExist(err):
		action.Kind = ActionCreate
	default:
		action.Kind = ActionConflict
		action.Content = []byte(fmt.Sprintf("could not inspect file: %v", err))
	}
	return action
}

func normalizedCleanDirs(dirs []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		clean := filepath.Clean(dir)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		normalized = append(normalized, clean)
	}
	return normalized
}

func isInsideAnyCleanDir(path string, cleanDirs []string) bool {
	cleanPath := filepath.Clean(path)
	for _, dir := range cleanDirs {
		rel, err := filepath.Rel(dir, cleanPath)
		if err != nil {
			continue
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return true
	}
	return false
}

func normalizedStrategy(strategy FileStrategy) FileStrategy {
	if strategy == "" {
		return StrategyCreateOnly
	}
	return strategy
}
