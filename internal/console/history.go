package console

import (
	"fmt"
	"io"
	"strings"

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

// WriteHistoryBlock writes every line with the persistent history marker.
func WriteHistoryBlock(out io.Writer, text string) error {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return WriteHistorySpace(out)
	}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			if err := WriteHistorySpace(out); err != nil {
				return err
			}
			continue
		}
		if err := WriteHistoryf(out, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
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

func (hw *HistoryWriter) WriteHistoryBlock(text string) *HistoryWriter {
	if hw.err != nil {
		return hw
	}

	hw.err = WriteHistoryBlock(hw.out, text)
	return hw
}

func (hw *HistoryWriter) Error() error {
	return hw.err
}
