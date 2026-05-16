package install

import "fmt"

// FormatSummary renders an install plan summary.
func FormatSummary(plan Plan) string {
	counts := map[ActionKind]int{}
	for _, action := range plan.Actions {
		counts[action.Kind]++
	}

	summary := fmt.Sprintf(
		"Create: %d\nUpdate: %d\nSkip: %d\nConflicts: %d\n",
		counts[ActionCreate],
		counts[ActionUpdate],
		counts[ActionSkip],
		counts[ActionConflict],
	)
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
