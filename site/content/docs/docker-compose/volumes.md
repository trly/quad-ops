---
title: "Volumes"
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

## Important Notes

1. **Volume Naming**: The volume name is used as the basis for the systemd unit name.

2. **Volume References**: When mounting a volume in a service, include the `.volume` suffix in the Podman unit file:
   ```ini
   # In the generated Podman container unit file
   Volume=data.volume:/data
   ```

3. **Volume Persistence**: Volumes persist independently of container lifecycles.

4. **Driver Support**: Only volume drivers supported by Podman can be used.

5. **Local Driver**: The `local` driver is the default if not specified.