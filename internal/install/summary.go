package install

import "fmt"

// FormatSummary renders an install plan summary.
func FormatSummary(plan Plan) string {
	counts := map[ActionKind]int{}
	for _, action := range plan.Actions {
		counts[action.Kind]++
	}

	summary := fmt.Sprintf(
		"Clean: %d\nCreate: %d\nUpdate: %d\nOverwrite: %d\nSkip: %d\nConflicts: %d\n",
		counts[ActionCleanDir],
		counts[ActionCreate],
		counts[ActionUpdate],
		counts[ActionOverwrite],
		counts[ActionSkip],
		counts[ActionConflict],
	)
	if counts[ActionOverwrite] > 0 {
		summary += "\nOverwrite files:\n"
		for _, action := range plan.Actions {
			if action.Kind == ActionOverwrite {
				summary += fmt.Sprintf("- %s\n", action.Path)
			}
		}
	}
	if counts[ActionConflict] > 0 {
		summary += "\nConflict files:\n"
		for _, action := range plan.Actions {
			if action.Kind == ActionConflict {
				summary += fmt.Sprintf("- %s\n", action.Path)
			}
		}
	}
	return summary
}
