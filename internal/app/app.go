package app

import (
	"context"
	"io"

	"github.com/cairon666/agentflow/internal/adapter"
	"github.com/cairon666/agentflow/internal/install"
)

// GitCloner clones a git repository into a destination directory.
type GitCloner interface {
	Clone(context.Context, string, string) error
}

// App owns the CLI use cases.
type App struct {
	Registry  adapter.Registry
	Writer    install.Writer
	Stdout    io.Writer
	WorkDir   string
	HomeDir   string
	GitCloner GitCloner
}
