---
title: "Docker Compose Volumes"
linkTitle: "Volumes"
weight: 20
description: >-
  Working with Docker Compose volumes in quad-ops
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

## Volume Configuration

quad-ops supports Docker Compose volume configurations and converts them to Podman quadlet volume units.

### Basic Volume Configuration

```yaml
volumes:
  data:
    driver: local
  db-data:
    driver: local
    driver_opts:
      type: btrfs
      device: /dev/sda1
```

### External Volumes

quad-ops supports the `external: true` directive for volumes. When a volume is marked as external, quad-ops will not create a quadlet unit for that volume, as it's assumed the volume is managed elsewhere.

```yaml
volumes:
  external_volume:
    external: true
```

When using external volumes:

1. quad-ops assumes that there is a matching systemd unit that can be referenced using `.volume` in container unit files
2. The external volume must exist before containers that depend on it are started
3. The volume can be managed by another quad-ops configuration or manually created

### Volume Options

Most Docker Compose volume options are supported and converted to their Podman quadlet equivalents:

- `driver`: Sets the volume driver (local, etc.)
- `driver_opts`: Converted to volume options
- `labels`: Custom metadata labels

### Referencing Volumes in Services

Volumes can be referenced in services like this:

```yaml
services:
  web:
    image: nginx
    volumes:
      - data:/var/www/html  # Named volume
      - /host/path:/container/path  # Bind mount
      - external_volume:/data  # External volume
```

In the container quadlet unit, these references will be converted to:

- Named volumes: `Volume=project-name-data.volume:/var/www/html`
- Bind mounts: `Volume=/host/path:/container/path`
- External volumes: `Volume=project-name-external_volume.volume:/data`