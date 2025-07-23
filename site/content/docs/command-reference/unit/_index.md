---
title: "unit"
weight: 40
---

# unit

Subcommands for managing and viewing quadlet units.

## Synopsis

```
quad-ops unit <subcommand> [OPTIONS]
```

## Description

The `unit` command provides management and inspection capabilities for Podman Quadlet units synchronized by quad-ops. It offers several subcommands to list, show, and check the status of managed units.

## Subcommands

- **[list](list)** - List units currently managed by quad-ops
- **[show](show)** - Show the contents of a quadlet unit

## Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |
| `--user` | `-u` | Run in rootless user mode |
| `--quadlet-dir` | | Override unit output directory |
| `--repository-dir` | | Override git checkout directory |

## Unit Types

The following unit types are supported:

| Type | Description |
|------|-------------|
| `container` | Podman container units (includes init containers) |
| `volume` | Podman volume units |
| `network` | Podman network units |
| `image` | Podman image units |
| `all` | All unit types |

## Related Commands

- **[sync](../sync)** - Synchronize repositories to create units
- **[up](../up)** - Start container units
- **[down](../down)** - Stop container units
- **[config](../config)** - Validate configuration

## See Also

- [Container Management](../../container-management) - Understanding unit lifecycle
- [Configuration](../../configuration) - Unit configuration options
- [Getting Started](../../getting-started) - Initial setup guide