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
| `--debug` | | Enable debug output |
| `--verbose` | | Enable verbose output |
| `--help` | `-h` | Show help information |

## Available Commands

### Command-Specific Help

```bash
# Help for any command
quad-ops <command> --help

# Examples
quad-ops sync --help
quad-ops validate --help
```

### Core Operations

- **[sync](sync)** - Sync repositories, generate Quadlet units, pull images, and start services
- **[validate](validate)** - Validate compose files for use with quad-ops
- **[update](update)** - Update quad-ops to the latest version
- **[version](version)** - Print version information

