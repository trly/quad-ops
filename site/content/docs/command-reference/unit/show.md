---
title: "show"
weight: 20
---

# unit show

Show the contents of a quadlet unit.

## Synopsis

```
quad-ops unit show <unit-name>
```

## Description

The `show` subcommand displays the contents of a specific quadlet unit file. This is useful for inspecting the configuration of units managed by quad-ops.

## Arguments

| Argument | Description |
|----------|-------------|
| `unit-name` | Name of the unit to display |

## Examples

```bash
# Show container unit contents
quad-ops unit show myapp-web

# Show volume unit contents  
quad-ops unit show myapp-data

# Show network unit contents
quad-ops unit show myapp-network
```

## Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | `-c` | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |

## Related Commands

- **[list](list)** - List units currently managed by quad-ops
- **[sync](../sync)** - Synchronize repositories to create units
- **[up](../up)** - Start container units
- **[down](../down)** - Stop container units