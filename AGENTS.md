# AGENTS.md - Development Guidelines for Quad-Ops

## Build/Test/Lint Commands
- **Build**: `task build` or `go build -o quad-ops cmd/quad-ops/main.go`
- **Test all**: `task test` or `gotestsum --format pkgname --format-icons text -- -coverprofile=coverage.out -v ./...`
- **Test single package**: `go test -v ./internal/compose`
- **Test single function**: `go test -v ./internal/compose -run TestProcessorBasic`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

## Architecture & Structure
- **CLI framework**: Cobra-based CLI with commands in `cmd/`
- **Core logic**: `internal/` packages: compose (Docker Compose processing), systemd (unit management), git (repo operations), config (configuration)
- **Main binary**: `cmd/quad-ops/main.go` - GitOps tool for converting Docker Compose to Podman Quadlet systemd units
- **Key packages**: `internal/compose` (compose processing), `internal/systemd` (systemd integration), `internal/repository` (git operations)

## Code Style & Conventions
- **Naming**: PascalCase exports, camelCase unexported, lowercase packages, descriptive variable names
- **Error handling**: Early returns, wrapped errors with `fmt.Errorf(..., %w, err)`, structured logging
- **Imports**: Standard lib first, third-party, then internal packages (`github.com/trly/quad-ops/internal/...`)
- **Patterns**: Interface-based design, dependency injection, factory functions (`NewX`), YAML-tagged structs for config
- **Testing**: Uses testify/assert, table-driven tests preferred

## Documentation
- <https://trly.github.io/quad-ops/> sources: @site/ 
- compose spec: <https://raw.githubusercontent.com/compose-spec/compose-spec/refs/heads/main/spec.md/>
- podman-systemd: <https://docs.podman.io/en/latest/_sources/markdown/podman-systemd.unit.5.md.txt/>

