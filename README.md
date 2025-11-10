# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg) ![GitHub License](https://img.shields.io/github/license/trly/quad-ops) ![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops) [![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

## GitOps for Quadlet

Quad-Ops is a lightweight GitOps framework for Podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). It watches Git repositories for standard Docker Compose files and automatically converts them into systemd unit files to run your containers.

**For comprehensive documentation, visit [https://trly.github.io/quad-ops/](https://trly.github.io/quad-ops/)**

## Key Features

- **GitOps workflow** - Monitor multiple Git repositories for container configurations
- **Standard Docker Compose** - Full support for Docker Compose files (services, networks, volumes, secrets)
- **Cross-platform** - Works on Linux (systemd/Quadlet) and macOS (launchd) with Podman
- **Smart change detection** - SHA256-based detection prevents unnecessary service restarts
- **Init containers** - Run initialization containers before main services start (similar to Kubernetes)
- **Intelligent restarts** - Only restarts services whose artifacts actually changed
- **Podman-specific features** - Support for exposing secrets as environment variables
- **Flexible deployment** - Works in both system-wide and user (rootless) modes
- **Production-ready** - Built with dependency injection, comprehensive test coverage (582+ tests)

## Compose Feature Support

Quad-Ops converts standard Docker Compose files to Podman Quadlet units. It supports all container runtime features that work with standalone Podman.

### Unsupported Fields

**Standard Compose fields (not yet implemented):**
- `volumes_from` - Use named volumes instead
- `stdin_open`, `tty` - Interactive mode not practical in systemd units
- `logging.driver` - Custom drivers (only journald, k8s-file, none, passthrough supported)

**Docker Swarm orchestration features (rejected with error):**
- `deploy.mode: global` - Use Kubernetes/Nomad for multi-node orchestration
- `deploy.replicas > 1` - Use Kubernetes/Nomad for multi-instance services
- `deploy.placement` - Use Kubernetes/Nomad for placement constraints
- `deploy.update_config`, `deploy.rollback_config` - Use Kubernetes/Nomad for rolling updates
- `deploy.endpoint_mode` - Use Kubernetes/Nomad for service discovery
- `ports.mode: ingress` - Use `mode: host` for Podman
- `secrets.driver` - Use file/content/environment sources instead of Swarm secret store

Use `quad-ops validate` to check your Compose files for unsupported features.

## Architecture

Quad-Ops uses a clean, modular architecture with clear separation of concerns:

```
Docker Compose → Service Specs → Platform Artifacts → Service Lifecycle
     ↓               ↓                 ↓                    ↓
   Reader        Converter         Renderer             Lifecycle
                                                         Manager
```

1. **Compose Reader** - Parses Docker Compose YAML files
2. **Spec Converter** - Converts to platform-agnostic service specifications
3. **Platform Renderer** - Generates platform-specific artifacts (Quadlet units on Linux, launchd plists on macOS)
4. **Lifecycle Manager** - Manages service start/stop/restart via systemd or launchd
5. **Change Detection** - SHA256 hashing ensures only changed services restart

This architecture makes it easy to:
- Add new platforms (Windows services, etc.)
- Test components in isolation with dependency injection
- Understand and maintain the codebase

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Configuration Example

```yaml
repositories:
  - name: quad-ops-compose  # Repository name (required)
    url: "https://github.com/example/repo.git"  # Git repository URL (required)
    ref: "main"  # Git reference to checkout: branch, tag, or commit hash (optional)
    composeDir: "compose"  # Subdirectory where Docker Compose files are located (optional)
```

## Getting Started with Development

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git
cd quad-ops

# Install task runner (if not already installed)
# macOS: brew install go-task/tap/go-task
# Linux: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

# Build, lint, test, and format (all-in-one)
task build

# Individual commands
task test          # Run all tests
task lint          # Run golangci-lint
task fmt           # Format code
go build -o quad-ops cmd/quad-ops/main.go  # Build binary only
```

## Installation

### Quick Install (Recommended)

Linux and macOS are both supported with automatic platform detection:

```bash
# System-wide installation (Linux and macOS)
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash

# User installation (rootless containers)
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --user
```

The installer automatically:
- Detects your platform (Linux/macOS) and architecture (amd64/arm64)
- Downloads and verifies the correct binary
- Installs systemd services (Linux only)
- Sets up example configuration files

### Manual Installation

```bash
# Build the binary
go build -o quad-ops cmd/quad-ops/main.go

# Move to system directory
sudo mv quad-ops /usr/local/bin/

# Copy the example config file
sudo mkdir -p /etc/quad-ops
sudo cp configs/config.yaml.example /etc/quad-ops/config.yaml

# Linux only: Install the systemd service file (optional)
sudo cp build/quad-ops.service /etc/systemd/system/quad-ops.service
sudo systemctl daemon-reload
sudo systemctl enable --now quad-ops
```
