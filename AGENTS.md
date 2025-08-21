# quad-ops Agent Guidelines

## Commands

- **Build**: `task build` (includes fmt, lint, test, build)
- **Test all**: `task test` or `go test -v ./...`
- **Test single**: `go test -v -run TestFunctionName ./internal/package/`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

## Architecture

GitOps framework converting Docker Compose to systemd Quadlet units:

- `cmd/` - CLI commands and main entry point
- `internal/compose/` - Docker Compose parsing and conversion
- `internal/unit/` - Quadlet unit generation (container, network, volume, build)
- `internal/systemd/` - systemd orchestration and service management
- `internal/git/` - Git repository synchronization
- `internal/config/` - Configuration management via Viper
- `internal/fs/` - File system operations with hash-based change detection
- `internal/execx/` - Command execution abstraction for testability
- `internal/testutil/` - Test utilities and helpers for reducing boilerplate
- `internal/validate/` - System validation with dependency injection

## Code Style

- Use testify (`assert`/`require`) for tests with descriptive names
- Package comments follow "Package X provides Y" format
- Interface-based design with default providers pattern
- Error handling with early returns, wrap with context
- Use structured logging via `slog`
- Test helpers: `testutil.NewTestLogger()`, `testutil.NewMockConfig()`,
temp dirs with cleanup
- Import grouping: stdlib, external, internal
- Constructor injection: Accept dependencies as parameters, no global state
- Command execution: Use `execx.Runner` interface for testable system commands
