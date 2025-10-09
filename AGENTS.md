# quad-ops Agent Guidelines

GitOps framework for Podman containers on Linux and macOS.

## Commands

- **Build**: `task build` (fmt, lint, test, compile)
- **Test all**: `task test` or `gotestsum --format pkgname --format-icons text -- -coverprofile=coverage.out -v ./...`
- **Test single**: `go test -run TestName ./path/to/package -v`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

## Architecture

Core pipeline: `Docker Compose → Spec Converter → Platform Renderer → Lifecycle Manager`.

### Key Packages

- `internal/compose/` - Parses Docker Compose files, converts to platform-agnostic service specs
- `internal/platform/systemd/` - Quadlet unit file renderer for Linux
- `internal/platform/launchd/` - Plist renderer for macOS
- `internal/service/` - Core service specification models (models.go, validate.go)
- `internal/repository/` - Unit file storage and git synchronization
- `internal/systemd/` - systemd/DBus integration for lifecycle management
- `cmd/` - CLI commands using Cobra (sync, up, down, daemon, etc.)

## Code Style

- **Testing**: table-driven tests preferred, heavy use of dependency injection and mocks
- **Imports**: Group stdlib, external packages, then internal (`github.com/trly/quad-ops/internal/*`)
- **Comments**: Package-level godoc required, exported functions documented
- **Error handling**: Wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Naming**: Service specs use sanitized names via `service.SanitizeName()`, unit files prefixed with project name
- **Validation**: All specs validated via `spec.Validate()` before rendering
- **Linters**: errcheck, govet, staticcheck, unused, revive, gosec, misspell enabled via golangci-lint

## Change Validation

Always Build after a set of changes are completed to ensure tests pass and the application has no compilation issues.
