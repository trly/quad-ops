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

## Compose Specification Support

Quad-Ops converts standard Docker Compose files to Podman Quadlet units. It supports all container runtime features that work with standalone Podman.

### Fully Supported

**Core container configuration:**
- `image`, `build`, `command`, `entrypoint`, `working_dir`, `user`, `hostname`

**Environment and labels:**
- `environment`, `env_file`, `labels`, `annotations`

**Networking:**
- `networks` (bridge, host, custom networks)
- `ports` (host mode only)
- `dns`, `dns_search`, `dns_opt`, `extra_hosts`
- `network_mode` (bridge, host, none, container:name)

**Storage:**
- `volumes` (bind mounts, named volumes, tmpfs)
- `secrets` with file/content/environment sources
- `configs` with file/content/environment sources

**Resources:**
- `memory`, `cpu_shares`, `cpu_quota`, `cpu_period`
- `pids_limit`, `shm_size`, `sysctls`, `ulimits`

**Security:**
- `cap_add`, `cap_drop`, `privileged`, `security_opt`, `read_only`
- `group_add`, `pid` mode, `ipc` mode, `cgroup_parent`

**Devices and hardware:**
- `devices`, `device_cgroup_rules`
- `runtime` (e.g., nvidia for GPU support)

**Health and lifecycle:**
- `healthcheck` (test, interval, timeout, retries, start_period)
- `restart` (maps to systemd restart policies)
- `stop_signal`, `stop_grace_period`
- `depends_on` (maps to systemd After/Requires)

### Partially Supported

**Secrets and configs:**
- File sources (`file: ./secret.txt`)
- Content sources (`content: "secret data"`)
- Environment sources (`environment: SECRET_VAR`)
- NOT supported: Swarm driver (`external: true` with `driver`)

**Resource constraints:**
- `deploy.resources.limits` (memory, cpus, pids)
- `deploy.resources.reservations` (partial - depends on cgroups v2)

**Dependency conditions:**
- All `depends_on` conditions (`service_started`, `service_healthy`, `service_completed_successfully`) map to systemd `After` + `Requires`
- No health-based startup gating (Quadlet limitation)

**Logging:**
- Supported: `journald`, `k8s-file`, `none`, `passthrough`
- NOT supported: Custom drivers not supported by Podman

### Not Supported - Use Alternatives

**Standard Compose fields:**
- `volumes_from` - Use named volumes or bind mounts
- `stdin_open`, `tty` - Interactive mode not practical in systemd units
- `extends` - Use YAML anchors or include directives

### Explicitly Out of Scope - Swarm Orchestration

Quad-Ops is **NOT** a Swarm orchestrator. These features are rejected with validation errors:

- `deploy.mode: global` - Multi-node replication
- `deploy.replicas > 1` - Multi-instance services
- `deploy.placement` - Node placement constraints
- `deploy.update_config`, `deploy.rollback_config` - Rolling updates
- `deploy.endpoint_mode` (vip/dnsrr) - Swarm service discovery
- `ports.mode: ingress` - Swarm load balancing (use `mode: host`)
- `configs`/`secrets` with `driver` field - Swarm secret store

**For these features, use:**
- **Kubernetes** - Cloud-native orchestration with full feature set
- **Nomad** - Lightweight orchestrator for VMs and containers
- **Docker Swarm** - If you need Swarm-specific features

Use `quad-ops validate` to check your Compose files for unsupported features.

**Reference:** [Podman Quadlet Documentation](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

## Naming Requirements

Quad-Ops enforces strict naming requirements for project and service names per Docker Compose specification.

### Project Names

Pattern: `^[a-z0-9][a-z0-9_-]*$`

- Must start with lowercase letter or digit
- Can contain only: lowercase letters, digits, dashes, underscores
- Examples: `myproject`, `my-project`, `my_project`, `project123`

### Service Names

Pattern: `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`

- Must start with alphanumeric character (upper or lowercase)
- Can contain: alphanumeric, dashes, underscores, periods
- Examples: `web`, `Web`, `web-api`, `web.api`, `Service123`

**Invalid names are rejected with clear error messages.**

## Cross-Project Dependencies

Quad-Ops supports declaring dependencies on services in other projects using the `x-quad-ops-depends-on` extension field.

### Example

```yaml
# project-app/compose.yml
name: app
services:
  backend:
    image: myapp:latest
    x-quad-ops-depends-on:
      - project: infrastructure
        service: proxy
        optional: false  # Fail if not found
      - project: monitoring
        service: prometheus
        optional: true   # Warn if not found
    depends_on:
      - redis  # Intra-project dependency
  
  redis:
    image: redis:latest
```

### How It Works

1. **Validation** - Quad-Ops validates that external services exist before deployment
2. **Ordering** - External dependencies are included in topological startup ordering
3. **Platform integration** - Maps to systemd `After`/`Requires` or launchd `DependsOn`
4. **Optional dependencies** - Can be marked as optional (warn if missing instead of failing)

**Requirements:**
- External service must already be deployed (use `quad-ops up` in dependency project first)
- Project and service names must follow naming requirements
- Works on both Linux (systemd) and macOS (launchd)

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
