package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/cairon666/agentflow/internal/adapter"
	"github.com/cairon666/agentflow/internal/adapter/claude"
	"github.com/cairon666/agentflow/internal/adapter/codex"
	"github.com/cairon666/agentflow/internal/adapter/opencode"
	"github.com/cairon666/agentflow/internal/app"
	"github.com/cairon666/agentflow/internal/install"
)

// NewRootCommand creates the agentflow command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "agentflow",
		Short:         "Generate agent CLI configuration from a portable template",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	root.SetHelpCommand(&cobra.Command{Use: "__help", Hidden: true})
	root.AddCommand(newUseCommand(newApp()))
	return root
}

func newApp() app.App {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return app.App{
		Registry: adapter.NewRegistry(
			codex.Adapter{},
			claude.Adapter{},
			opencode.Adapter{},
		),
		Writer:  install.NewWriter(),
		Stdout:  os.Stdout,
		WorkDir: workDir,
		HomeDir: homeDir,
	}
}
