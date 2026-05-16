package console

import (
	"errors"
	"testing"
)

func TestReporterPanicsOnWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	reporter := NewReporter(errorWriter{err: writeErr})
	tests := []struct {
		name string
		call func()
	}{
		{name: "banner", call: reporter.Banner},
		{name: "history", call: func() { reporter.Historyf("Template: %s\n", "test") }},
		{name: "history space", call: reporter.HistorySpace},
		{name: "message", call: func() { reporter.Message("hello") }},
		{name: "message line", call: func() { reporter.MessageLine("hello") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Fatal("expected panic")
				}
				err, ok := recovered.(error)
				if !ok {
					t.Fatalf("panic = %T %v, want error", recovered, recovered)
				}
				if !errors.Is(err, writeErr) {
					t.Fatalf("panic = %v, want wrapped %v", err, writeErr)
				}
			}()

			tt.call()
		})
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
