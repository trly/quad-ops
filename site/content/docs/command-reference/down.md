---
title: "down"
weight: 25
---

# quad-ops down

Stop managed services by stopping their systemd units.

## Synopsis

```
quad-ops down [project] [flags]
```

## Arguments

```
  [project]   Optional project name to stop (stops all if not specified)
```

## Options

```
  -s, --service strings   Specific service(s) to stop within the project
  -h, --help              help for down
```

## Global Options

```
    --config string   Path to the configuration file
    --debug           Enable debug mode
    --verbose         Enable verbose output
```

## Description

The `down` command stops managed container services via systemd D-Bus.

## Examples

### Stop all services

```bash
quad-ops down
```

### Stop services for a specific project

```bash
quad-ops down myproject
```

### Stop specific services within a project

```bash
quad-ops down myproject -s web -s api
```

### Stop with verbose output

```bash
quad-ops down --verbose
```
