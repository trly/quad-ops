---
title: "Quad-Ops Configuration"
weight: 10
---

# Quad-Ops Configuration

Quad-Ops uses a YAML configuration file to define global settings and repository management options.

## Configuration File Location

By default, Quad-Ops looks for configuration files in the following locations:

- `/etc/quad-ops/config.yaml` (system-wide)
- `~/.config/quad-ops/config.yaml` (user mode)

You can specify a custom configuration file using the `--config` flag:

```bash
quad-ops --config /path/to/config.yaml sync
```

## Global Settings

### Core Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where Git repositories are cloned |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for Podman Quadlet unit files |



## Rootless (User Mode) Configuration

For rootless operation as a non-root user, Quad-Ops automatically adjusts default paths based on the effective UID:

| Setting | System Mode (root) | User Mode (non-root) |
|---------|-------------|-----------|
| `repositoryDir` | `/var/lib/quad-ops` | `~/.local/share/quad-ops` |
| `quadletDir` | `/etc/containers/systemd` | `~/.config/containers/systemd` |

To override these defaults, explicitly set the paths in your configuration file.

## Example Configuration

### Minimal Configuration

```yaml
repositories:
  - name: myapp
    url: https://github.com/user/myapp.git
```

### Complete Configuration

```yaml
# Global settings
repositoryDir: /var/lib/quad-ops
quadletDir: /etc/containers/systemd

# Repository definitions
repositories:
  - name: webapp
    url: https://github.com/company/webapp.git
    ref: main
    composeDir: deploy

  - name: microservices
    url: https://github.com/company/microservices.git
    ref: production
    composeDir: compose
```

### Environment-Specific Configuration

```yaml
# Development environment
repositories:
  - name: dev-app
    url: https://github.com/company/app.git
    ref: develop
```

```yaml
# Production environment
repositories:
  - name: prod-app
    url: https://github.com/company/app.git
    ref: v2.1.0  # Pin to specific version
```

## Configuration Validation

Quad-Ops validates the configuration file on startup and will report errors for:

- Invalid YAML syntax
- Missing required fields

## Next Steps

- [Repository Configuration](repository-configuration) - Learn about repository-specific options
- [Quick Start](../quick-start/) - Set up your first Quad-Ops deployment
