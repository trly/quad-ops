---
title: "sync"
weight: 10
---

# quad-ops sync

Synchronizes Docker Compose files from configured repositories with Podman Quadlet units on the local system.

Repositories are defined in the quad-ops config file as a list of Repository objects.

---

```yaml
repositories:
  - name: quad-ops-compose
    url: https://github.com/trly/quad-ops-compose.git
    ref: main
```

## Synopsis

```
quad-ops sync [flags]
```

## Options

```
      --rollback   Rollback to the previous sync state
  -h, --help       help for sync
```

## Global Options

```
    --config string   Path to the configuration file
    --debug           Enable debug mode
    --verbose         Enable verbose output
```

## Description

The `sync` command is the core operation of Quad-Ops. It performs a complete synchronization cycle:

1. **Repository Updates** - Clone new repositories or pull latest changes
2. **File Discovery** - Scan for Docker Compose files in configured locations
3. **Conversion** - Generate Podman Quadlet units from compose configurations
4. **Deployment** - Write units to the quadlet directory

This command is safe to run repeatedly and will only make necessary changes.

Use `--rollback` to revert to the previous sync state if something goes wrong.

## Examples

### Synchronize all configured repositories

```bash
quad-ops sync
```

### Synchronize with verbose output

```bash
quad-ops sync --verbose
```

### Synchronize with custom config file

```bash
quad-ops sync --config /path/to/config.yaml
```
