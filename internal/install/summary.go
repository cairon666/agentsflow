package install

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// FormatSummary renders an install plan summary.
func FormatSummary(plan Plan) string {
	counts := countActions(plan.Actions)
	summary := formatActionCounts(counts)
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

// FormatDryRunPreview renders a detailed install plan preview without applying it.
func FormatDryRunPreview(plan Plan) string {
	return FormatSummary(plan) + FormatDryRunFilePreview(plan)
}

// FormatDryRunFilePreview renders the file detail section of a dry-run preview.
func FormatDryRunFilePreview(plan Plan) string {
	var b strings.Builder
	writeFilePreviews(&b, plan.Actions)
	return b.String()
}

func countActions(actions []Action) map[ActionKind]int {
	counts := map[ActionKind]int{}
	for _, action := range actions {
		counts[action.Kind]++
	}
	return counts
}

func formatActionCounts(counts map[ActionKind]int) string {
	return fmt.Sprintf(
		"Create: %d\nUpdate: %d\nOverwrite: %d\nSkip: %d\nConflicts: %d\n",
		counts[ActionCreate],
		counts[ActionUpdate],
		counts[ActionOverwrite],
		counts[ActionSkip],
		counts[ActionConflict],
	)
}

func writeFilePreviews(b *strings.Builder, actions []Action) {
	wroteHeading := false
	for _, action := range actions {
		if action.Kind == ActionCleanDir {
			continue
		}
		if !wroteHeading {
			b.WriteString("\nFiles:\n")
			wroteHeading = true
		}
		writeFilePreview(b, action)
	}
}

func writeFilePreview(b *strings.Builder, action Action) {
	fmt.Fprintf(b, "%s %s (%s)\n", fileActionPrefix(action.Kind), action.Path, action.Kind)
	if writesFile(action.Kind) || action.Kind == ActionSkip {
		writeContentBlock(b, action.Content)
	}
	if action.Kind == ActionUpdate && action.Strategy == StrategyMerge {
		writeMergeDiff(b, action)
	}
}

func writesFile(kind ActionKind) bool {
	return kind == ActionCreate || kind == ActionUpdate || kind == ActionOverwrite
}

func fileActionPrefix(kind ActionKind) string {
	switch kind {
	case ActionCreate:
		return "+"
	case ActionUpdate, ActionOverwrite:
		return "+/-"
	case ActionSkip:
		return "="
	case ActionConflict:
		return "!"
	default:
		return "?"
	}
}

func writeContentBlock(b *strings.Builder, content []byte) {
	b.WriteString("--- planned content ---\n")
	if len(content) > 0 {
		b.Write(content)
		if content[len(content)-1] != '\n' {
			b.WriteString("\n")
		}
	}
	b.WriteString("--- end planned content ---\n")
}

func writeMergeDiff(b *strings.Builder, action Action) {
	diff, err := unifiedDiff(action.Path, action.ExistingContent, action.Content)
	if err != nil || strings.TrimSpace(diff) == "" {
		return
	}
	b.WriteString("--- merge diff ---\n")
	b.WriteString(diff)
	if diff[len(diff)-1] != '\n' {
		b.WriteString("\n")
	}
}

func unifiedDiff(path string, existing, planned []byte) (string, error) {
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        diffLines(existing),
		B:        diffLines(planned),
		FromFile: path + " (current)",
		ToFile:   path + " (planned)",
		Context:  3,
	})
}

func diffLines(content []byte) []string {
	if len(content) == 0 {
		return nil
	}
	text := string(content)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return lines
}
