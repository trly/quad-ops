---
title: "Configuration"
weight: 20
bookCollapseSection: true
---

# Configuration

This section covers all configuration options for Quad-Ops, from basic setup to advanced repository management.

## Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where repositories are stored |
| `syncInterval` | duration | `5m` | Interval between repository synchronization |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for quadlet files |
| `userMode` | boolean | `false` | Whether to run in user mode |
| `verbose` | boolean | `false` | Enable verbose logging |
| `usePodmanDefaultNames` | boolean | `false` | Whether to use Podman's default container naming with systemd- prefix |
| `repositories` | array | - | List of repositories to manage |

## Repository Options
| Option | Type | Default | Description |
|-------------------|------|---------|-------------|
| `name` | string | - | Unique identifier for the repository |
| `url` | string | - | Git repository URL to clone/pull from |
| `ref` | string | - | Git reference to checkout (branch, tag, or commit hash) |
| `composeDir` | string | "" | Subdirectory within repo where Docker Compose files are located |
| `usePodmanDefaultNames` | boolean | `false` | Whether to use Podman's default naming for this repository (overrides global setting) |

## Example Configuration

```yaml
# Global settings
repositoryDir: /var/lib/quad-ops
syncInterval: 10m
quadletDir: /etc/containers/systemd
userMode: false
verbose: true
usePodmanDefaultNames: false  # No systemd- prefix in container hostnames

# Repository definitions
repositories:
  - name: app1
    url: https://github.com/example/app1
    ref: main
    composeDir: compose

  - name: app2
    url: https://github.com/example/app2
    ref: dev
    usePodmanDefaultNames: true  # Use systemd- prefix in container hostnames for this repo only
```