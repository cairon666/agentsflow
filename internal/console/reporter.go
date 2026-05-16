package console

import (
	"context"
	"fmt"
	"io"

	"github.com/cairon666/agentsflow/internal/ui/terminal"
)

// Reporter writes terminal output for CLI use cases.
type Reporter struct {
	out io.Writer
}

// NewReporter creates a terminal reporter.
func NewReporter(out io.Writer) Reporter {
	if out == nil {
		out = io.Discard
	}
	return Reporter{out: out}
}

// Banner writes the application banner.
func (r Reporter) Banner() {
	mustWrite(WrintBanner(r.out))
}

// Historyf writes a persistent history line.
func (r Reporter) Historyf(format string, args ...any) {
	mustWrite(WriteHistoryf(r.out, format, args...))
}

// HistoryBlock writes every line as persistent history.
func (r Reporter) HistoryBlock(text string) {
	mustWrite(WriteHistoryBlock(r.out, text))
}

// HistorySpace writes a persistent history spacer line.
func (r Reporter) HistorySpace() {
	mustWrite(WriteHistorySpace(r.out))
}

// Message writes user-facing text without a trailing newline.
func (r Reporter) Message(args ...any) {
	_, err := fmt.Fprint(r.out, args...)
	mustWrite(err)
}

// MessageLine writes user-facing text with a trailing newline.
func (r Reporter) MessageLine(args ...any) {
	_, err := fmt.Fprintln(r.out, args...)
	mustWrite(err)
}

// RunLoading runs an action with terminal loading feedback.
func (r Reporter) RunLoading(ctx context.Context, title string, action func(context.Context) error) error {
	return terminal.RunWithLoading(ctx, r.out, title, action)
}

func mustWrite(err error) {
	if err != nil {
		panic(err)
	}
}
