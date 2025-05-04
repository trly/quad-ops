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

A lightweight GitOps framework for podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

Quad-Ops is a tool that helps you manage container deployments using Podman and systemd in a GitOps workflow. It watches Git repositories for standard Docker Compose files and automatically converts them into unit files that systemd can use to run your containers.

## What Makes Quad-Ops Different

While Quad-Ops uses Docker Compose as its configuration format, there are some key differences from traditional Docker Compose deployments:

1. **GitOps-Based**: Changes to containers are driven by Git repositories, not manual commands
2. **Systemd Integration**: Containers are managed by systemd instead of a Docker daemon
3. **Podman Backend**: Uses Podman's daemonless container engine instead of Docker
4. **Automated Dependencies**: Service relationships are automatically converted to systemd unit dependencies
5. **Intelligent Restarts**: Only restarts services that have changed and their dependents

## Key Features:
- Monitor multiple Git repositories for container configurations
- Supports standard Docker Compose files (services, networks, volumes, secrets)
- Works in both system-wide and user (rootless) modes
- Automates deployment and management of container infrastructure


## Feature Support

Quad-Ops supports Docker Compose version 3.x files with comprehensive feature coverage, only limited by what is currently supported by the Podman systemd integration.

| Supported Features | Unsupported Features |
|--------------------|----------------------|
| ✅ **Container Configuration** | ❌ **Privileged Mode** |
| ✅ **Container Relationships** | ❌ **Security Labels** |
| ✅ **Secrets (File & Environment)** | ❌ **DNS Configuration in Networks** |
| ✅ **Networks with Aliases** | ❌ **Swarm Mode** |
| ✅ **Volumes** | ❌ **Docker-specific Extensions** |
| ✅ **Resource Dependencies** | |
| ✅ **Podman Extensions** | |
| ✅ **Health Checks** | |

### Podman-Specific Extensions

Quad-Ops supports several Podman-specific extensions through Docker Compose extension fields:

| Extension | Description |
|-----------|-------------|
| `x-podman-env-secrets` | Maps secrets to environment variables instead of files |

### Container Naming

Quad-Ops provides two container naming modes controlled by the `usePodmanDefaultNames` option:

- **Default (false)**: Container hostnames match their service names without the systemd- prefix
  - Example: `myapp-db` for a service named "db" in project "myapp"
- **Podman Default (true)**: Container hostnames use Podman's default naming with systemd- prefix 
  - Example: `systemd-myapp-db` for the same service

