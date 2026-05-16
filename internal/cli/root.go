package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/cairon666/agentsflow/internal/composition"
)

// Version is set at build time for release binaries.
var Version = "dev"

// NewRootCommand creates the agentsflow command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "agentsflow",
		Short:         "Generate agent CLI configuration from a portable template",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.SetHelpCommand(&cobra.Command{Use: "__help", Hidden: true})
	root.AddCommand(newUseCommand(composition.NewApp(composition.Config{Stdout: os.Stdout})))
	return root
}
