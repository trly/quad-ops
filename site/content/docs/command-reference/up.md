---
title: "up"
weight: 20
---

# quad-ops up

Start managed services by pre-pulling container images and starting systemd units.

## Synopsis

```
quad-ops up [project] [flags]
```

## Arguments

```
  [project]   Optional project name to start (starts all if not specified)
```

## Options

```
  -s, --service strings   Specific service(s) to start within the project
  -h, --help              help for up
```

## Global Options

```
    --config string   Path to the configuration file
    --debug           Enable debug mode
    --verbose         Enable verbose output
```

## Description

The `up` command starts managed container services. It:

1. **Discovers services** from compose files in configured repositories
2. **Pre-pulls images** to avoid timeouts during service start
3. **Starts services** via systemd D-Bus

Services with missing Podman secrets are skipped with a warning.

## Examples

### Start all services

```bash
quad-ops up
```

### Start services for a specific project

```bash
quad-ops up myproject
```

### Start specific services within a project

```bash
quad-ops up myproject -s web -s api
```

### Start with verbose output

```bash
quad-ops up --verbose
```
