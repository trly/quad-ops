---
title: "up"
weight: 20
---

# up

Start all managed container units.

## Synopsis

```
quad-ops up
```

## Description

The `up` command starts all container units that have been synchronized from configured repositories. It performs the following operations:

1. **Database Query** - Retrieves all container units from the quad-ops database
2. **Unit Reset** - Resets any failed units before attempting to start them
3. **Service Start** - Starts each container unit using systemd
4. **Status Report** - Provides feedback on successful and failed operations

This command is useful for bringing up your entire container infrastructure after system restarts or maintenance.

## Options

No command-specific options are available for this command.

### Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | `-c` | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |

## Examples

```bash
# Start all managed containers
quad-ops up
```

## Related Commands

- **[sync](sync)** - Synchronize repositories before starting
- **[down](down)** - Stop all managed containers
- **[unit list](unit-list)** - Check container status after starting
- **[unit show](unit-show)** - View detailed unit configuration

## See Also

- [Container Management](../container-management) - Understanding container lifecycle
- [Getting Started](../getting-started) - Initial setup guide