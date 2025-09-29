---
title: "up"
weight: 20
---

# quad-ops up

Start managed units synchronized from repositories.

If no unit names are provided, starts all managed units.
If unit names are provided, starts only the specified units.

Examples:
  quad-ops up                    # Start all units
  quad-ops up web-service        # Start specific unit
  quad-ops up web api database   # Start multiple units

## Synopsis

```
quad-ops up [unit-name...] [flags]
```

## Options

```
  -h, --help   help for up
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

The `up` command starts container units that have been synchronized from configured repositories. It performs the following operations:

1. **Unit Discovery** - Finds all container units in the quadlet directory (or specific named units)
2. **Unit Reset** - Resets any failed units before attempting to start them
3. **Service Start** - Starts each container unit using systemd
4. **Status Report** - Provides feedback on successful and failed operations

This command is useful for bringing up your entire container infrastructure after system restarts or maintenance.

## Examples

### Start all managed containers

```bash
quad-ops up
```

### Start a specific service

```bash
quad-ops up web-service
```

### Start multiple specific services

```bash
quad-ops up web api database
```

### Start with verbose output

```bash
quad-ops up --verbose
```

### Start in user mode

```bash
quad-ops up --user
```

### Start with JSON output format

```bash
quad-ops up --output json
```

