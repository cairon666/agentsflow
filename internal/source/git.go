package source

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitCLICloner clones repositories by invoking the git CLI.
type GitCLICloner struct{}

// Clone clones source into dest with a shallow checkout.
func (GitCLICloner) Clone(ctx context.Context, source, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", source, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("clone template repository: %w: %s", err, message)
		}
		return fmt.Errorf("clone template repository: %w", err)
	}
	return nil
}
