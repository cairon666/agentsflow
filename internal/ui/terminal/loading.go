package terminal

import (
	"context"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/x/term"
)

func RunWithLoading(ctx context.Context, out io.Writer, title string, action func(context.Context) error) error {
	if out == nil {
		out = io.Discard
	}

	if !isTerminalWriter(out) {
		if _, err := fmt.Fprintln(out, title); err != nil {
			return err
		}
		return action(ctx)
	}

	return spinner.New().
		Title(title).
		WithTheme(spinner.ThemeFunc(func(bool) *spinner.Styles {
			return spinner.ThemeDefault(true)
		})).
		WithViewHook(func(v tea.View) tea.View {
			v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, 1)
			return v
		}).
		WithOutput(out).
		ActionWithErr(func(context.Context) error {
			return action(ctx)
		}).
		Run()
}

func isTerminalWriter(out io.Writer) bool {
	file, ok := out.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(file.Fd())
}
