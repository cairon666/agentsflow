package install

import (
	"bytes"
	"fmt"
	"os"
)

// BuildPlan resolves raw desired file contents into create/update/skip/conflict actions.
func BuildPlan(target, scope string, desired map[string][]byte) Plan {
	actions := make([]Action, 0, len(desired))
	for path, content := range desired {
		actions = append(actions, classifyAction(path, content, false))
	}
	return Plan{Target: target, Scope: scope, Actions: actions}
}

// BuildPlanWithManagedPaths resolves desired files and treats selected paths as
// safe to update.
func BuildPlanWithManagedPaths(target, scope string, desired map[string][]byte, managedPaths map[string]struct{}) Plan {
	actions := make([]Action, 0, len(desired))
	for path, content := range desired {
		_, forcedManaged := managedPaths[path]
		actions = append(actions, classifyAction(path, content, forcedManaged))
	}
	return Plan{Target: target, Scope: scope, Actions: actions}
}

func classifyAction(path string, content []byte, managed bool) Action {
	action := Action{Path: path, Content: content, ManagedBy: managed}
	existing, err := os.ReadFile(path)
	switch {
	case err == nil && bytes.Equal(existing, content):
		action.Kind = ActionSkip
	case err == nil && managed:
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
