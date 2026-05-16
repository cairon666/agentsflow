package install

import (
	"bytes"
	"fmt"
	"os"
)

// BuildPlan resolves rendered artifacts into create/update/skip/conflict actions.
func BuildPlan(artifacts ArtifactSet) Plan {
	actions := make([]Action, 0, len(artifacts.Files))
	for _, file := range artifacts.Files {
		actions = append(actions, classifyAction(file.Path, file.Content, normalizedStrategy(file.Strategy)))
	}
	return Plan{Target: artifacts.Target, Scope: artifacts.Scope, Actions: actions}
}

func classifyAction(path string, content []byte, strategy FileStrategy) Action {
	action := Action{Path: path, Content: content, Strategy: strategy}
	existing, err := os.ReadFile(path)
	switch {
	case err == nil && bytes.Equal(existing, content):
		action.Kind = ActionSkip
	case err == nil && strategy == StrategyMerge:
		action.Kind = ActionUpdate
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

func normalizedStrategy(strategy FileStrategy) FileStrategy {
	if strategy == "" {
		return StrategyCreateOnly
	}
	return strategy
}
