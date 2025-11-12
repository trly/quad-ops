---
title: "quad-ops"
weight: 0
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# ![quad-ops](images/quad-ops-64.png) quad-ops

## GitOps for Quadlet
![GitHub License](https://img.shields.io/github/license/trly/quad-ops)
![Docs Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml)
![Build Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/docs.yaml?label=docs)
![CodeQL Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml?label=codeql)
![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops)
[![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

A cross-platform GitOps framework for container management with native service integration

Quad-Ops is a tool that helps you manage container deployments in a GitOps workflow. It watches Git repositories for standard [Docker Compose](https://compose-spec.io/) files and automatically converts them into native service definitions for your platform:

- **Linux**: systemd + [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- **macOS**: launchd (planned)

## What Makes Quad-Ops Different

While Quad-Ops uses [Docker Compose](https://compose-spec.io/)  as its configuration format, there are some key differences from traditional [Docker Compose](https://compose-spec.io/)  deployments:

1. **GitOps-Based**: Changes to containers are driven by Git repositories, not manual commands
2. **Cross-Platform**: Automatically adapts to your platform's native service manager (systemd on Linux, launchd on macOS)
3. **Native Integration**: Containers are managed by your platform's service manager, not a separate daemon
4. **Platform-Agnostic Models**: Uses platform-neutral service definitions that render to platform-specific formats
5. **Automated Dependencies**: Service relationships are automatically converted to native dependency directives
6. **Intelligent Restarts**: Only restarts services that have changed and their dependents

## Key Features:
- Monitor multiple Git repositories for container configurations
- Supports standard [Docker Compose](https://compose-spec.io/)  files (services, networks, volumes, secrets)
- Works in both system-wide and user (rootless) modes
- Automates deployment and management of container infrastructure


## How Quad-Ops Works

Quad-Ops bridges the gap between [Docker Compose](https://compose-spec.io/)  and systemd by converting familiar [Docker Compose](https://compose-spec.io/)  configurations into [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) units:

```mermaid
graph LR
    A[Git Repository] --> B[Docker Compose Files]
    B --> C[Quad-Ops Conversion]
    C --> D[Podman Quadlet Units]
    D --> E[systemd Services]
    E --> F[Running Containers]
```

### The Conversion Process

1. **Git Synchronization** - Monitors repositories for [Docker Compose](https://compose-spec.io/)  file changes
2. **File Processing** - Reads `docker-compose.yml` files and associated environment files
3. **Unit Generation** - Converts services, volumes, and networks to `.container`, `.volume`, and `.network` Quadlet units
4. **systemd Integration** - Loads units into systemd for native service management
5. **Dependency Resolution** - Maps `depends_on` relationships to systemd `After`/`Requires` directives

### Why This Approach?

- **Familiar Configuration** - Use standard [Docker Compose](https://compose-spec.io/)  files you already know
- **systemd Benefits** - Leverage systemd's robust service management, logging, and dependency handling
- **GitOps Workflow** - All changes tracked in Git with rollback capability
- **Podman Integration** - Daemonless, rootless container execution with enhanced security

## Docker Compose Feature Support

Quad-Ops converts [Docker Compose](https://compose-spec.io/)  version 3.x+ configurations into systemd-managed containers through [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). The following matrix shows which [Docker Compose](https://compose-spec.io/)  features are supported and how they're implemented.

### Feature Support Matrix

This matrix describes [Docker Compose](https://compose-spec.io/)  feature support for **systemd-managed containers** through Quad-Ops conversion:

---
**✅ Native Quadlet Support** - Features that map directly to [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) directives for optimal systemd integration.

**⚠️ PodmanArgs Implementation** - Features implemented using Quadlet's `PodmanArgs` directive, providing full functionality with some limitations.

**❌ Unsupported** - Docker-specific features incompatible with systemd container management.

---
| Category | Feature | Support Level | Implementation Method |
|----------|---------|---------------|----------------------|
| **Containers** | Image/Command | ✅ Full | Native Quadlet directives |
| | Ports | ✅ Full | Native Quadlet directives |
| | Volumes | ✅ Full | Native Quadlet directives |
| | Environment | ✅ Full | Native Quadlet directives |
| | Restart Policy | ✅ Full | Native Quadlet directives |
| | Dependencies | ✅ Full | systemd After/Requires |
| **Resources** | Memory Limits | ⚠️ Partial | PodmanArgs implementation |
| | CPU Limits | ⚠️ Partial | PodmanArgs implementation |
| | PIDs Limit | ⚠️ Partial | PodmanArgs implementation |
| **Security** | Privileged Mode | ⚠️ Partial | PodmanArgs implementation |
| | Capabilities | ⚠️ Partial | PodmanArgs implementation |
| | SELinux Labels | ⚠️ Partial | PodmanArgs implementation |
| **Networking** | Custom Networks | ✅ Full | Native Quadlet directives |
| | Network Aliases | ✅ Full | Native Quadlet directives |
| | DNS Configuration | ⚠️ Partial | PodmanArgs implementation |
| **Health Checks** | Health Commands | ⚠️ Partial | Native Quadlet directives |
| | Health Dependencies | ❌ None | Not supported in systemd |
| **Build** | Dockerfile Builds | ✅ Full | Quadlet Build units |
| | Build Arguments | ✅ Full | Quadlet Build units |
| | Multi-stage Builds | ✅ Full | Quadlet Build units |

## Compose Extensions

Quad-Ops extends [Docker Compose](https://compose-spec.io/)  with powerful capabilities:

- **Cross-project dependencies** (`x-quad-ops-depends-on`) - Declare dependencies on services in other projects
- **Environment secrets** mapping for secure credential handling
- **Strict naming validation** - Project and service names follow Docker Compose specification exactly
- **Volume extensions** for advanced mount options
- **Build extensions** for enhanced build configurations

See [Docker Compose Support](docs/container-management/docker-compose-support) for complete configuration examples and [Cross-Project Dependencies](docs/container-management/cross-project-dependencies) for multi-project architectures.
