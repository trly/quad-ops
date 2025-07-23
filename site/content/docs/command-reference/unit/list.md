---
title: "list"
weight: 10
---

# unit list

List units currently managed by quad-ops.

## Synopsis

```
quad-ops unit list [OPTIONS]
```

## Description

The `list` subcommand displays all quadlet units that are currently managed by quad-ops. You can filter by unit type to view specific categories of units.

## Options

| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| `--type` | `-t` | string | `container` | Type of unit to list (container, volume, network, image, all) |

## Examples

```bash
# List all container units (default)
quad-ops unit list

# List all units of all types
quad-ops unit list --type all

# List only volume units
quad-ops unit list --type volume

# List only network units
quad-ops unit list --type network
```

## Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |
| `--user` | `-u` | Run in rootless user mode |
| `--quadlet-dir` | | Override unit output directory |
| `--repository-dir` | | Override git checkout directory |

## Related Commands

- **[show](show)** - Show the contents of a quadlet unit
- **[sync](../sync)** - Synchronize repositories to create units
- **[up](../up)** - Start container units
- **[down](../down)** - Stop container units