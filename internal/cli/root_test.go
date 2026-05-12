package cli

import "testing"

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
