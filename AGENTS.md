# quad-ops Agent Guidelines

GitOps framework for podman containers on Linux and macOS.

## Commands

- **Build**: `task build` (fmt, lint, test, compile)
- **Test all**: `task test` or `gotestsum --format pkgname --format-icons text -- -coverprofile=coverage.out -v ./...`
- **Test single**: `go test -run TestName ./path/to/package -v`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

**Compile only**: `go build -o quad-ops ./cmd/quad-ops`

## Project Structure

```
cmd/quad-ops/     # CLI entrypoint (kong-based)
internal/
  compose/        # Docker Compose file parsing (compose-spec/compose-go)
  config/         # Application configuration
  git/            # Git repository operations (go-git)
  state/          # Deployment state persistence (commit tracking for rollback)
  systemd/        # Quadlet unit file generation
configs/          # Example configuration files
site/             # Hugo documentation site
```

## Key Dependencies

- `github.com/alecthomas/kong` - CLI framework
- `github.com/compose-spec/compose-go/v2` - Docker Compose parsing
- `github.com/coreos/go-systemd/v22` - systemd D-Bus integration
- `github.com/creativeprojects/go-selfupdate` - Self-update mechanism
- `github.com/go-git/go-git/v5` - Git operations
- `github.com/stretchr/testify` - Testing assertions
- `gopkg.in/ini.v1` - INI file handling (Quadlet unit generation)

## Code Conventions

- Use `testify/assert` and `testify/require` for test assertions
- Internal packages follow standard Go project layout
- Quadlet unit files are systemd-compatible `.container`, `.network`, `.volume` files
- Support both system-wide (`/etc/containers/systemd`) and user (`~/.config/containers/systemd`) modes
