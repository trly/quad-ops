# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg) ![GitHub License](https://img.shields.io/github/license/trly/quad-ops) ![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops) [![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

## GitOps for Quadlet

Quad-Ops is a lightweight GitOps framework for Podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). It watches Git repositories for standard Docker Compose files and automatically converts them into systemd unit files to run your containers.

**For comprehensive documentation, visit [https://trly.github.io/quad-ops/](https://trly.github.io/quad-ops/)**

## Key Features

- Monitor multiple Git repositories for container configurations
- Support for standard Docker Compose files (services, networks, volumes, secrets)
- Support for Podman-specific features like exposing secrets as environment variables
- **Init containers** - Run initialization containers before main services start (similar to Kubernetes)
- Automatic detection of service-specific environment files
- Automated dependencies via systemd unit relationships
- Intelligent restarts - only restarts services that changed and their dependents
- Works in both system-wide and user (rootless) modes

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

```bash
# Build the binary
go build -o quad-ops cmd/quad-ops/main.go

# Move to system directory
sudo mv quad-ops /usr/local/bin/

# Copy the example config file
sudo mkdir -p /etc/quad-ops
sudo cp configs/config.yaml.example /etc/quad-ops/config.yaml

# Install the systemd service file (optional)
sudo cp build/quad-ops.service /etc/systemd/system/quad-ops.service

# Reload systemd daemon
sudo systemctl daemon-reload

# Enable and start the service
sudo systemctl enable --now quad-ops
```
