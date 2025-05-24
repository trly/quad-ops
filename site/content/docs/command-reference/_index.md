---
title: "Command Reference"
weight: 50
bookCollapseSection: true
---

# Command Reference

Complete reference for all Quad-Ops commands with detailed options, examples, and use cases.

## Command Structure

Quad-Ops follows a hierarchical command structure:

```
quad-ops [global-options] <command> [command-options] [arguments]
```

### Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | `-c` | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |
| `--help` | `-h` | Show help information |
| `--version` | | Show version information |

## Available Commands

### Core Operations
- **[sync](sync)** - Synchronize repositories and deploy containers
- **[up](up)** - Start all or specific services
- **[down](down)** - Stop and remove services
- **[update](update)** - Update quad-ops to the latest version

### Unit Management
- **[unit list](unit-list)** - List deployed units and their status
- **[unit show](unit-show)** - Display detailed unit information
- **[unit remove](unit-remove)** - Remove specific units

### Configuration
- **[config](config)** - Configuration management commands

## Command Categories

### Repository Operations
Commands that interact with Git repositories and perform synchronization.

### Service Management
Commands for controlling container lifecycle and examining running services.

### Unit Administration
Commands for managing Quadlet units and their systemd integration.

### System Configuration
Commands for validating and managing Quad-Ops configuration.

## Common Usage Patterns

### Initial Deployment
```bash
# Configure repositories and perform first sync
sudo quad-ops sync
```

### Regular Operations
```bash
# Check service status
sudo quad-ops unit list

# Restart specific service
sudo quad-ops up myapp-web

# Stop all services for maintenance
sudo quad-ops down
```

## Exit Codes

Quad-Ops uses standard exit codes for scripting and automation:

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid command usage |
| `3` | Configuration error |
| `4` | Git operation failed |
| `5` | systemd operation failed |

## Getting Help

### Command-Specific Help
```bash
# Help for any command
quad-ops <command> --help

# Examples
quad-ops sync --help
quad-ops unit list --help
```

### Manual Pages
```bash
# View manual page (if installed)
man quad-ops
man quad-ops-sync
```

## Next Steps

Browse the individual command references for detailed information on options, examples, and use cases.