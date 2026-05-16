package terminal

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunWithLoadingUsesAccessibleModeForNonTerminalOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := RunWithLoading(t.Context(), &stdout, "Loading repository...", func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Loading repository") {
		t.Fatalf("stdout missing loading title:\n%s", output)
	}
	if strings.Contains(output, "[?2026") || strings.Contains(output, "[?2027") || strings.Contains(output, "]11;") {
		t.Fatalf("stdout included terminal query sequences:\n%q", output)
	}
}
