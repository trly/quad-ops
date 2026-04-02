# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg) ![GitHub License](https://img.shields.io/github/license/trly/quad-ops) ![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops) [![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

## GitOps for Quadlet

Quad-Ops is a lightweight GitOps framework for Podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). It watches Git repositories for standard Docker Compose files and automatically converts them into systemd unit files to run your containers.

**For comprehensive documentation, visit [https://trly.github.io/quad-ops/](https://trly.github.io/quad-ops/)**

## Getting Started with Development

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git
cd quad-ops

# Install task runner (if not already installed)
# Linux: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

# Build, lint, test, and format (all-in-one)
task build

# Individual commands
task test          # Run all tests
task lint          # Run golangci-lint
task fmt           # Format code
go build -o quad-ops ./cmd/quad-ops  # Build binary only
```
