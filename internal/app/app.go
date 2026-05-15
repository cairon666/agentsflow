package app

import (
	"io"

	"github.com/cairon666/agentsflow/internal/adapter"
	"github.com/cairon666/agentsflow/internal/install"
	"github.com/cairon666/agentsflow/internal/source"
)

// App owns the CLI use cases.
type App struct {
	Registry       adapter.Registry
	Writer         install.Writer
	SourceResolver source.Resolver
	Stdout         io.Writer
	WorkDir        string
	HomeDir        string
}
