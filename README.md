# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg) ![GitHub License](https://img.shields.io/github/license/trly/quad-ops) ![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops) [![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

## Description

Quad-Ops is a lightweight GitOps command-line tool for Podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). It watches Git repositories for standard Docker Compose files and automatically converts them into systemd Quadlet unit files (`.container`, `.network`, `.volume`) to run your containers. It works in both system-wide (`/etc/containers/systemd`) and user/rootless (`~/.config/containers/systemd`) modes.

It is intended for operators who want to manage Podman workloads declaratively from Git without adopting a full orchestrator.

**For comprehensive documentation, visit [https://trly.github.io/quad-ops/](https://trly.github.io/quad-ops/)**

The CLI provides the following commands (see `cmd/quad-ops/`):

- `quad-ops sync` - sync repositories, write systemd unit files, and start services
- `quad-ops validate` - validate compose files for use with quad-ops
- `quad-ops update` - update quad-ops to the latest version
- `quad-ops version` - print version information

### Key Features

- **GitOps workflow** - Monitor multiple Git repositories for container configurations
- **Standard Docker Compose** - Full support for Docker Compose files (services, networks, volumes, secrets)
- **Quadlet integration** - Generates native systemd Quadlet unit files
- **Flexible deployment** - Works in both system-wide and user (rootless) modes
- **Validation** - Check Compose files for compatibility before deployment

## Setup

Quad-Ops is written in Go (`go 1.25.9`, see `go.mod` and `.mise.toml`) and targets Linux and macOS.

### Install from releases

```bash
# System-wide install
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash

# User install to ~/.local/bin
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --user
```

### Build from source

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git
cd quad-ops

# Install the task runner (if not already installed)
# Linux: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

# Build, lint, vulncheck, test, and compile (all-in-one)
task build

# Individual commands
task test          # Run all tests (gotestsum)
task lint          # Run golangci-lint
task fmt           # Format code
go build -o quad-ops ./cmd/quad-ops  # Compile binary only
```

Development toolchain versions are pinned in `.mise.toml` (Go, golangci-lint, gotestsum, hugo-extended, node, pnpm, pre-commit, task).

### Configuration

Quad-Ops loads configuration from a YAML file. The default path depends on mode:

- System mode: `/etc/quad-ops/config.yaml`
- User mode: `~/.config/quad-ops/config.yaml`

Pass an explicit path with `--config`. A reference file is provided at `configs/config.yaml.example`.

## System context

| Component | Relationship | Evidence |
| --- | --- | --- |
| Podman / Quadlet | Downstream runtime; generated systemd unit files are executed by Podman via Quadlet | `internal/systemd/`, `AGENTS.md` |
| systemd | Downstream; Quad-Ops manages units over the systemd D-Bus API | `github.com/coreos/go-systemd/v22` in `go.mod` |
| Git repositories | Upstream source of truth; cloned/pulled for Compose definitions | `internal/git/`, `github.com/go-git/go-git/v5` in `go.mod` |
| Docker Compose spec | Input contract parsed into Quadlet units | `internal/compose/`, `github.com/compose-spec/compose-go/v2` in `go.mod` |
| GitHub releases | Source for the `update` self-update mechanism | `cmd/quad-ops/update.go`, `github.com/creativeprojects/go-selfupdate` in `go.mod` |

## Compose Specification Support

Quad-Ops converts standard Docker Compose files to Podman Quadlet units. It supports all container runtime features that work with standalone Podman.

### Fully Supported

**Core container configuration:**
- `image`, `build`, `command`, `entrypoint`, `working_dir`, `hostname`

**Environment and labels:**
- `environment`, `env_file`, `labels`, `annotations`

**Networking:**
- `networks` (bridge, host, custom networks)
- `ports` (host mode only)
- `dns`, `dns_search`, `dns_opt`, `extra_hosts`
- `network_mode` (bridge, host)

**Storage:**
- `volumes` (bind mounts, named volumes)
- `secrets` with file/content/environment sources

**Resources:**
- `memory`, `cpus`, `cpu_shares`, `cpuset`
- `pids_limit`, `shm_size`, `sysctls`, `ulimits`

**Security:**
- `cap_add`, `cap_drop`, `privileged`, `security_opt`, `read_only`
- `group_add`, `pid` mode, `ipc` mode (private, shareable)

**Devices and hardware:**
- `devices`

**Health and lifecycle:**
- `healthcheck` (test, interval, timeout, retries, start_period)
- `restart` (maps to systemd restart policies)
- `stop_signal` (SIGTERM, SIGKILL), `stop_grace_period`
- `depends_on` with `service_started` condition (maps to systemd After/Requires)
- `tty`, `stdin_open`, `init`, `pull_policy`

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
- `depends_on` with `service_started` maps to systemd `After` + `Requires`
- NOT supported: `service_healthy`, `service_completed_successfully` conditions

**Logging:**
- Supported: `json-file`, `journald`
- NOT supported: Other logging drivers

### Not Supported - Use Alternatives

**Standard Compose fields:**
- `user` - Use systemd user mapping instead
- `tmpfs` - Use named volumes, bind mounts, or `x-quad-ops-mounts`
- `volumes_from` - Use named volumes or bind mounts
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
