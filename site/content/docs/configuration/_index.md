---
title: "Configuration"
weight: 20
bookCollapseSection: true
---

# Configuration

This section covers all configuration options for Quad-Ops, from basic setup to advanced repository management.

## How It Works

Quad-Ops provides a GitOps approach to container management:

1. **Git synchronization** pulls the latest changes from configured repositories
2. **File discovery** recursively locates Docker Compose files
3. **Conversion** generates Podman Quadlet unit files (`.container`, `.network`, `.volume`)
4. **Deployment** loads systemd services for container lifecycle management

All changes are driven by Git commits — providing version control, rollback capability, and an audit trail for infrastructure changes. Use `quad-ops validate` in CI/CD pipelines for pre-deployment validation.

Generated Quadlet units integrate with systemd for dependency management (`After`/`Requires` directives), automatic restart on failure, and logging through journald.

## Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where repositories are stored |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for quadlet files |
| `repositories` | array | - | List of repositories to manage |

## Repository Options
| Option | Type | Default | Description |
|-------------------|------|---------|-------------|
| `name` | string | - | Unique identifier for the repository |
| `url` | string | - | Git repository URL to clone/pull from |
| `ref` | string | remote HEAD | Git reference to checkout (branch, tag, or commit hash) |
| `composeDir` | string | "" | Subdirectory within repo where Docker Compose files are located |

## Example Configuration

```yaml
# Global settings
repositoryDir: /var/lib/quad-ops
quadletDir: /etc/containers/systemd

# Repository definitions
repositories:
  - name: app1
    url: https://github.com/example/app1
    ref: main
    composeDir: compose

  - name: app2
    url: https://github.com/example/app2
    ref: dev
```
