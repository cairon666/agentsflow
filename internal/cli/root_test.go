package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandExposesOnlyUse(t *testing.T) {
	cmd := NewRootCommand()
	public := []string{}
	for _, child := range cmd.Commands() {
		if child.Hidden {
			continue
		}
		public = append(public, child.Name())
	}
	if len(public) != 1 || public[0] != "use" {
		t.Fatalf("public commands = %v, want [use]", public)
	}
}

func TestRootCommandPrintsVersion(t *testing.T) {
	previous := Version
	Version = "1.2.3-test"
	t.Cleanup(func() {
		Version = previous
	})

	cmd := NewRootCommand()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "1.2.3-test" {
		t.Fatalf("version output = %q, want %q", got, "1.2.3-test")
	}
}
