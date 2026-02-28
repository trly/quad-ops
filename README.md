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

