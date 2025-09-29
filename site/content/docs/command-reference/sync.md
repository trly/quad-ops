---
title: "sync"
weight: 10
---

# quad-ops sync

Synchronizes the Docker Compose files defined in configured repositories with quadlet units on the local system.

Repositories are defined in the quad-ops config file as a list of Repository objects.

---

```yaml
repositories:
  - name: quad-ops-compose
    url: https://github.com/trly/quad-ops-compose.git
    target: main
    cleanup:
      action: Delete
```

## Synopsis

```
quad-ops sync [flags]
```

## Options

```
  -d, --dry-run       Perform a dry run without making any changes.
  -f, --force         Force synchronization even if the repository has not changed.
  -h, --help          help for sync
  -r, --repo string   Synchronize a single, named, repository.

```

## Global Options

```
      --config string           Path to the configuration file
  -o, --output string           Output format (text, json, yaml) (default "text")
      --quadlet-dir string      Path to the quadlet directory
      --repository-dir string   Path to the repository directory
  -u, --user                    Run in user mode
  -v, --verbose                 Enable verbose logging
```

## Description

The `sync` command is the core operation of Quad-Ops. It performs a complete synchronization cycle:

1. **Repository Updates** - Clone new repositories or pull latest changes
2. **File Discovery** - Scan for Docker Compose files in configured locations
3. **Conversion** - Generate Podman Quadlet units from compose configurations
4. **Deployment** - Load units into systemd and start services
5. **Cleanup** - Remove outdated units that are no longer defined in any repository

This command is safe to run repeatedly and will only make necessary changes.

## Examples

### Synchronize all configured repositories

```bash
quad-ops sync
```

### Dry run to see what would be changed

```bash
quad-ops sync --dry-run
```

### Force synchronization of all repositories

```bash
quad-ops sync --force
```

### Synchronize only a specific repository

```bash
quad-ops sync --repo quad-ops-compose
```

### Synchronize with verbose output

```bash
quad-ops sync --verbose
```

### User mode synchronization

```bash
quad-ops sync --user
```

