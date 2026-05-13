## Commands

- Prefer Taskfile commands over raw `go` commands.
- Use `task fmt` to format code.
- Use `task tidy` after changing Go dependencies.
- Use `task lint` for static checks.
- Use `task test` for the default test suite.
- Use `task test:race` when concurrency, shared state, or CLI flow changes need stronger validation.
- Use `task check` before finalizing meaningful code changes.
- Use `task build` to build the CLI binary into `./bin/agentsflow`.
