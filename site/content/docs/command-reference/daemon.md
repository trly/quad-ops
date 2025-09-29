---
title: "daemon"
weight: 30
---

# quad-ops daemon

Run quad-ops as a daemon with periodic synchronization of configured repositories.

The daemon will perform initial synchronization and then continue running, periodically syncing repositories at the specified interval. This is ideal for continuous deployment scenarios where you want automatic updates.

The daemon integrates with systemd, sending readiness and watchdog notifications when running under systemd supervision.

## Synopsis

```
quad-ops daemon [flags]
```

## Options

```
  -f, --force                    Force synchronization even if repository has not changed
  -h, --help                     help for daemon
  -r, --repo string              Synchronize a single, named, repository
  -i, --sync-interval duration   Interval between synchronization checks (default 5m0s)
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

## Examples

### Run daemon with default 5-minute sync interval

```bash
quad-ops daemon
```

### Run daemon with custom sync interval

```bash
quad-ops daemon --sync-interval 10m
```

### Run daemon for a specific repository

```bash
quad-ops daemon --repo my-app-repo
```

### Force synchronization even if repository hasn't changed

```bash
quad-ops daemon --force
```

## systemd Integration

The daemon command is designed to work seamlessly with systemd:

- Sends readiness notifications when startup is complete
- Supports watchdog functionality for health monitoring
- Handles SIGTERM gracefully for clean shutdown
- Logs to systemd journal when running as a service
