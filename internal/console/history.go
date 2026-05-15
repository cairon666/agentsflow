package console

import (
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
)

var historyMarker = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("┃")

// WriteHistoryf writes persistent choice history using the same marker style as huh.
func WriteHistoryf(out io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(out, "%s  "+format, append([]any{historyMarker}, args...)...)
	return err
}

// WriteHistorySpace writes a blank persistent history line with only the marker.
func WriteHistorySpace(out io.Writer) error {
	_, err := fmt.Fprintln(out, historyMarker)
	return err
}

type HistoryWriter struct {
	out io.Writer
	err error
}

func NewHistoryWriter(out io.Writer) *HistoryWriter {
	return &HistoryWriter{out: out}
}

func (hw *HistoryWriter) WriteHistoryf(format string, args ...any) *HistoryWriter {
	if hw.err != nil {
		return hw
	}

	hw.err = WriteHistoryf(hw.out, format, args...)
	return hw
}

func (hw *HistoryWriter) WriteHistorySpace() *HistoryWriter {
	if hw.err != nil {
		return hw
	}

	hw.err = WriteHistorySpace(hw.out)
	return hw
}

func (hw *HistoryWriter) WriteHistoryBlockf(format string, args ...any) *HistoryWriter {
	if hw.err != nil {
		return hw
	}

	return hw.WriteHistorySpace().WriteHistoryf(format, args...).WriteHistorySpace()
}

func (hw *HistoryWriter) Error() error {
	return hw.err
}
