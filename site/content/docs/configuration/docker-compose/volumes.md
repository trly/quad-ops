---
title: "Volumes"
weight: 10
---

# Volumes

Volumes in Docker Compose are converted to Podman volume units. This allows you to define persistent storage for your containers.

## Supported Properties

- `name`: Volume name
- `driver`: Volume driver
- `driver_opts`: Driver options
- `labels`: Volume labels

## Example

```yaml
volumes:
  web-data:
    driver: local
    driver_opts:
      type: "nfs"
      o: "addr=192.168.1.1,rw"
      device: ":/path/to/dir"
    labels:
      environment: "production"
      usage: "web-content"
  
  db-data:
    driver: local
    labels:
      backup: "daily"
```

## Conversion to Podman Volume Units

When Quad-Ops processes a volume definition from a Docker Compose file, it creates a corresponding Podman volume unit with the following mapping:

| Docker Compose Property | Podman Volume Property |
|-------------------------|------------------------|
| `name` | `VolumeName` |
| `driver` | `Driver` |
| `driver_opts` | `Options` (each key-value pair becomes an option) |
| `labels` | `Label` |

The volume name is used as the basis for the systemd unit name, and the volume is managed through Podman's volume commands.