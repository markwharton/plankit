# CLAUDE.md — Go Extensions

Go-specific guidelines to add to your project's CLAUDE.md alongside the base template.

## Dependencies and Architecture

- Zero external dependencies — stdlib only.
- Subcommand routing via `os.Args` switch + `flag.FlagSet` per command.
- Dependency injection via Config structs for testability.

## Stdout/Stderr Separation

- All user messages to stderr. Stdout reserved for protocol/data output (JSON, etc.).

## Version and Build

- Version injection via `-ldflags "-X .../version.Version=x.y.z"`.
- Cross-compile targets via `make build-all` (darwin/linux/windows, amd64/arm64).
- Makefile targets: `build`, `build-all`, `test`, `lint`, `install`, `fmt`.

## Testing

- Tests use `t.TempDir()` for filesystem isolation.
- Use `httptest.NewServer` for HTTP mocks.
- Run with `-race` flag.

## File Operations

- Atomic writes: temp file + rename, with `.bak` backup for config files.
