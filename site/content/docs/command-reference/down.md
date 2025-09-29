---
title: "down"
weight: 30
---

# quad-ops down

Stop managed units synchronized from repositories.

If no unit names are provided, stops all managed units.
If unit names are provided, stops only the specified units.

Examples:
  quad-ops down                    # Stop all units
  quad-ops down web-service        # Stop specific unit
  quad-ops down web api database   # Stop multiple units

## Synopsis

```
quad-ops down [unit-name...] [flags]
```

## Options

```
  -h, --help   help for down
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

The `down` command stops container units that have been synchronized from configured repositories. It performs the following operations:

1. **Unit Discovery** - Finds all container units in the quadlet directory (or specific named units)
2. **Service Stop** - Stops each container unit using systemd
3. **Status Report** - Provides feedback on successful and failed operations

This command is useful for shutting down your entire container infrastructure for maintenance or system shutdown.

## Examples

### Stop all managed containers

```bash
quad-ops down
```

### Stop a specific service

```bash
quad-ops down web-service
```

### Stop multiple specific services

```bash
quad-ops down web api database
```

### Stop with verbose output

```bash
quad-ops down --verbose
```

### Stop in user mode

```bash
quad-ops down --user
```

### Stop with JSON output format

```bash
quad-ops down --output json
```

