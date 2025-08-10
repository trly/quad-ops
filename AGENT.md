# Agent Guidelines for quad-ops

## Project Overview
- **Quad-Ops**: GitOps framework for Podman containers managed by Quadlet
- Converts Docker Compose files to systemd unit files for container management
- Supports both system-wide and rootless container operation modes
- Written in Go with CLI interface using Cobra framework

## Architecture & Structure
- **cmd/**: CLI commands and main entry points
- **internal/**: Core application logic (compose, config, git, systemd, etc.)
- **configs/**: Configuration file examples
- **examples/**: Example configurations and compose files
- **site/**: Documentation site (Hugo-based)
- **build/**: Build artifacts and systemd service files

## Key Commands
- `task build` - Build application (includes fmt, lint, test)
- `task test` - Run tests with coverage
- `task lint` - Run linter
- `go test -v ./...` - Run all tests
- `go test -v -race ./...` - Run all tests with race detection (matches CI)

## Configuration
- Main config: `/etc/quad-ops/config.yaml` (system) or `~/.config/quad-ops/config.yaml` (user)
- Example config: `configs/config.yaml.example`
- Supports multiple Git repositories with Docker Compose files
- Profile-specific configurations for different environments

## Dependencies & Tools
- **Go** (managed via mise)
- **mise**: Development environment manager
- **task**: Task runner (Taskfile.yml)
- **golangci-lint**: Go linter with comprehensive rule set
- **gotestsum**: Enhanced test runner with formatting

## Code Style Guidelines
- Follow standard Go conventions
- Use golangci-lint rules (errcheck, govet, staticcheck, revive, gosec, etc.)
- Format code with gofmt and goimports
- Comprehensive test coverage expected
- Security-focused development (gosec linter enabled)

## Release & Distribution
- Automated releases via GoReleaser (`.goreleaser.yml`)
- Self-update capability built into application
- Systemd service file provided for daemon operation
- Installation script available (`install.sh`)

## Documentation
- Documentation site details and Hugo-specific guidelines: see `site/AGENT.md`

