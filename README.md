# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg) ![GitHub License](https://img.shields.io/github/license/trly/quad-ops) ![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops) [![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

## GitOps for Quadlet

Quad-Ops is a lightweight GitOps framework for Podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). It watches Git repositories for standard Docker Compose files and automatically converts them into systemd unit files to run your containers.

**For comprehensive documentation, visit [https://trly.github.io/quad-ops/](https://trly.github.io/quad-ops/)**

## Key Features

- **GitOps workflow** - Monitor multiple Git repositories for container configurations
- **Standard Docker Compose** - Full support for Docker Compose files (services, networks, volumes, secrets)
- **Quadlet integration** - Generates native systemd Quadlet unit files
- **Flexible deployment** - Works in both system-wide and user (rootless) modes
- **Validation** - Check Compose files for compatibility before deployment

## CLI Commands

```bash
quad-ops sync          # Sync repositories and write systemd unit files to quadlet directory
quad-ops validate      # Validate compose files for use with quad-ops
quad-ops dependencies  # Print dependencies for all repositories in the configuration
quad-ops update        # Update quad-ops to the latest version
quad-ops version       # Print version information
```

### Global Options

```
--config    Path to config file
--debug     Enable debug mode
--verbose   Enable verbose output
```

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

## Architecture

Quad-Ops uses a modular architecture:

1. **Compose Reader** - Parses Docker Compose YAML files
2. **Platform Renderer** - Generates Quadlet unit files for systemd

## Configuration Example

```yaml
syncInterval: "5m"           # How often to sync repositories (optional)
repositoryDir: "/var/lib/quad-ops"  # Where to clone repositories (optional)
quadletDir: "/etc/containers/systemd"  # Where to write Quadlet units (optional)

repositories:
  - name: my-containers       # Repository name (required)
    url: "https://github.com/example/repo.git"  # Git repository URL (required)
    ref: "main"               # Git reference: branch, tag, or commit (optional)
    composeDir: "compose"     # Subdirectory with Compose files (optional)
```

Default directories:
- **System mode** (root): `/var/lib/quad-ops` for repos, `/etc/containers/systemd` for Quadlet units
- **User mode** (rootless): `~/.local/share/quad-ops` for repos, `~/.config/containers/systemd` for Quadlet units

## Getting Started

```bash
# Sync repositories and generate Quadlet unit files
quad-ops sync

# Validate compose files before deployment
quad-ops validate /path/to/compose.yml

# Reload systemd and start services
systemctl daemon-reload
systemctl start <service-name>
```

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
go build -o quad-ops cmd/quad-ops/main.go  # Build binary only
```

## Installation

### Quick Install (Recommended)

```bash
# System-wide installation
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash

# User installation (rootless containers)
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --user
```

The installer automatically:
- Detects your architecture (amd64/arm64)
- Downloads and verifies the correct binary
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
```
