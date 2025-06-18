---
title: "down"
weight: 30
---

# down

Stop all managed container units.

## Synopsis

```
quad-ops down
```

## Description

The `down` command stops all container units that have been synchronized from configured repositories. It performs the following operations:

1. **Unit Discovery** - Finds all container units in the quadlet directory
2. **Service Stop** - Stops each container unit using systemd
3. **Status Report** - Provides feedback on successful and failed operations

This command is useful for shutting down your entire container infrastructure for maintenance or system shutdown.

## Options

No command-specific options are available for this command.

### Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | `-c` | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |

## Examples

```bash
# Stop all managed containers
quad-ops down
```

## Related Commands

- **[up](up)** - Start all managed containers
- **[sync](sync)** - Synchronize repositories
- **[unit list](unit-list)** - Check container status after stopping
- **[unit show](unit-show)** - View detailed unit configuration

## See Also

- [Container Management](../container-management) - Understanding container lifecycle
- [Getting Started](../getting-started) - Initial setup guide