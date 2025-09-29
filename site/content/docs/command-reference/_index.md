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
| `--config` | | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |
| `--user` | `-u` | Run in rootless user mode |
| `--quadlet-dir` | | Override unit output directory |
| `--repository-dir` | | Override git checkout directory |
| `--help` | `-h` | Show help information |

## Available Commands

### Command-Specific Help

```bash
# Help for any command
quad-ops <command> --help

# Examples
quad-ops sync --help
quad-ops unit list --help
```

### Core Operations

- **[sync](sync)** - Synchronize repositories and deploy containers
- **[daemon](daemon)** - Run quad-ops as a daemon with periodic synchronization
- **[up](up)** - Start all or specific services
- **[down](down)** - Stop and remove services
- **[validate](validate)** - Validate Docker Compose files and quad-ops extensions
- **[update](update)** - Update quad-ops to the latest version
- **[version](version)** - Show version information and check for updates

### System Health

- **[doctor](doctor)** - Check system health and configuration

### Image Management

- **[image pull](image)** - Pull container images from repositories

### Unit Management

- **[unit list](unit/list)** - List deployed units and their status
- **[unit show](unit/show)** - Display detailed unit information

### Configuration

- **[config](config)** - Configuration management commands

