---
title: "sync"
weight: 10
---

# sync

Synchronize Git repositories and deploy container configurations.

## Synopsis

```
quad-ops sync [OPTIONS]
```

## Description

The `sync` command is the core operation of Quad-Ops. It performs a complete synchronization cycle:

1. **Repository Updates** - Clone new repositories or pull latest changes
2. **File Discovery** - Scan for Docker Compose files in configured locations
3. **Conversion** - Generate Podman Quadlet units from compose configurations
4. **Deployment** - Load units into systemd and start services
5. **Cleanup** - Remove outdated units that are no longer defined in any repository

This command is safe to run repeatedly and will only make necessary changes.

## Options

| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| `--dry-run` | `-d` | boolean | `false` | Perform a dry run without making any changes |
| `--daemon` | | boolean | `false` | Run as a daemon |
| `--sync-interval` | `-i` | duration | `5m` | Interval between synchronization checks |
| `--repo` | `-r` | string | | Synchronize a single, named, repository |
| `--force` | `-f` | boolean | `false` | Force synchronization even if the repository has not changed |

### Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |
| `--user` | `-u` | Run in rootless user mode |
| `--quadlet-dir` | | Override unit output directory |
| `--repository-dir` | | Override git checkout directory |

## Related Commands

- **[up](up)** - Start services after sync
- **[down](down)** - Stop services before maintenance
- **[unit list](unit-list)** - Check sync results
- **[config](config)** - Validate configuration before sync

## See Also

- [Configuration](../configuration) - Setup and repository configuration
- [Getting Started](../getting-started) - Initial setup guide
- [Container Management](../container-management) - Understanding the sync process