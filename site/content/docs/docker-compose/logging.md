---
title: "Logging"
weight: 60
---

# Logging Configuration

Quad-Ops supports Docker Compose logging configuration, mapping it to Podman Quadlet logging directives.

## Logging Drivers

Podman (and Quad-Ops) supports several logging drivers that can be specified with the `log_driver` property:

- `journald`: Log to the systemd journal
- `json-file`: JSON formatted output files
- `k8s-file`: Format logs as Kubernetes Pod logs
- `passthrough`: Pass container logs directly to stdout/stderr 
- `syslog`: Print to system logger (syslog)

## Logging Options

The `log_opt` property allows you to configure logging driver options:

### Common Options

| Option | Description |
|--------|-------------|
| `max-size` | Maximum size of the log before rotation (e.g. "10m") |
| `max-file` | Maximum number of log files to retain after rotation |
| `tag` | Specify log tag (default: container name) |

### JSON File Options

| Option | Description |
|--------|-------------|
| `compress` | Enable compression of rotated log files |
| `path` | Specify custom log path instead of default |

### Journald Options

| Option | Description |
|--------|-------------|
| `tag` | Specify log tag (default: container name) |
| `priority` | Specify log priority level (e.g. "info", "debug") |

## Example

```yaml
services:
  web:
    image: docker.io/nginx:latest
    log_driver: json-file
    log_opt:
      max-size: "10m"
      max-file: "3"
      compress: "true"

  api:
    image: docker.io/myapi:latest
    log_driver: journald
    log_opt:
      tag: "api-service"
      priority: "info"
```

## Implementation Details

When Quad-Ops processes Docker Compose logging configuration, it maps the `log_driver` and `log_opt` properties directly to Podman Quadlet's equivalent `LogDriver` and `LogOpt` directives. These directives are included in the generated container unit files that systemd uses to manage the containers.