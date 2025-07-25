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
- `./config.yaml` (current directory)

You can specify a custom configuration file using the `--config` flag:

```bash
quad-ops --config /path/to/config.yaml sync
```

## Global Settings

### Core Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where Git repositories are cloned |
| `syncInterval` | duration | `5m` | Interval between automatic repository synchronization |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for Podman Quadlet unit files |
| `userMode` | boolean | `false` | Enable user-mode (rootless) operation |
| `verbose` | boolean | `false` | Enable verbose logging output |
| `unitStartTimeout` | duration | `10s` | Timeout for systemd unit start operations |
| `imagePullTimeout` | duration | `30s` | Timeout for systemd units during image pull phases (sync operations) |



## User Mode Configuration

For rootless operation, user mode changes several default paths:

| Setting | System Mode | User Mode |
|---------|-------------|-----------|
| `repositoryDir` | `/var/lib/quad-ops` | `~/.local/share/quad-ops` |
| `quadletDir` | `/etc/containers/systemd` | `~/.config/containers/systemd` |

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
syncInterval: 10m
quadletDir: /etc/containers/systemd
userMode: false
verbose: true
unitStartTimeout: 15s
imagePullTimeout: 60s

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
syncInterval: 1m
verbose: true

repositories:
  - name: dev-app
    url: https://github.com/company/app.git
    ref: develop
```

```yaml
# Production environment
syncInterval: 30m
verbose: false

repositories:
  - name: prod-app
    url: https://github.com/company/app.git
    ref: v2.1.0  # Pin to specific version
```

## Configuration Validation

Quad-Ops validates the configuration file on startup and will report errors for:

- Invalid YAML syntax
- Missing required fields
- Invalid duration formats
- Duplicate repository names

## Next Steps

- [Repository Configuration](repository-configuration) - Learn about repository-specific options
- [Getting Started](../getting-started) - Set up your first Quad-Ops deployment