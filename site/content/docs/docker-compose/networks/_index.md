---
title: "Networks"
---

# Networks

Networks in Docker Compose are converted to Podman network units. This allows you to create custom networks for your containers to communicate on.

## Supported Properties

- `driver`: Network driver
- `driver_opts`: Driver options
- `ipam`: IP address management configuration
  - `subnet`: Subnet in CIDR format
  - `gateway`: Gateway address
  - `ip_range`: Range of IPs
- `internal`: Internal network flag
- `enable_ipv6`: Enable IPv6 flag
- `labels`: Network labels

## Example

```yaml
networks:
  frontend:
    driver: bridge
    driver_opts:
      com.docker.network.bridge.name: front-bridge
    labels:
      environment: production
      tier: frontend

  backend:
    driver: bridge
    internal: true
    ipam:
      driver: default
      config:
        - subnet: 172.16.238.0/24
          gateway: 172.16.238.1
          ip_range: 172.16.238.0/24
    enable_ipv6: true
```

## Conversion to Podman Network Units

When Quad-Ops processes a network definition from a Docker Compose file, it creates a corresponding Podman network unit with the following mapping:

| Docker Compose Property | Podman Network Property |
|-------------------------|-------------------------|
| `driver` | `Driver` |
| `ipam.config[0].subnet` | `Subnet` |
| `ipam.config[0].gateway` | `Gateway` |
| `ipam.config[0].ip_range` | `IPRange` |
| `internal` | `Internal` |
| `enable_ipv6` | `IPv6` |
| `driver_opts` | `Options` (each key-value pair becomes an option) |
| `labels` | `Label` |

## Important Notes

1. **Network Naming**: The network name is used as the basis for the systemd unit name.

2. **DNS Resolution**: By default, DNS is enabled for Podman networks, allowing containers to resolve each other by name.

3. **Network References**: When connecting a container to a network, include the `.network` suffix in the Podman unit file:
   ```ini
   # In the generated Podman container unit file
   Network=frontend.network
   ```

4. **Default Network**: If no networks are defined, a default network named `<project-name>-default.network` is automatically created and all services will connect to it.

5. **Driver Support**: Only network drivers supported by Podman can be used.

6. **DNSEnabled**: This property is not supported by Podman Quadlet. Configure DNS via driver options instead.

7. **External Networks**: Networks marked with `external: true` are not created by Quad-Ops as they are expected to be managed elsewhere. These can be referenced by containers but must exist before the container starts.
