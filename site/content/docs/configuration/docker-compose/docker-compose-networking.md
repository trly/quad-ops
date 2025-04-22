---
title: "Docker Compose Networking"
linkTitle: "Networking"
weight: 30
description: >-
  Working with Docker Compose networks in quad-ops
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

## Network Configuration

quad-ops supports Docker Compose network configurations and converts them to Podman quadlet network units.

### Basic Network Configuration

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
          gateway: 172.20.0.1
```

### External Networks

quad-ops supports the `external: true` directive for networks. When a network is marked as external, quad-ops will not create a quadlet unit for that network, as it's assumed the network is managed elsewhere.

```yaml
networks:
  external_network:
    external: true
```

When using external networks:

1. quad-ops assumes that there is a matching systemd unit that can be referenced using `.network` in container unit files
2. The external network must exist before containers that depend on it are started
3. The network can be managed by another quad-ops configuration or manually created

### Network Options

Most Docker Compose network options are supported and converted to their Podman quadlet equivalents:

- `driver`: Sets the network driver (bridge, macvlan, etc.)
- `driver_opts`: Converted to network options
- `ipam.config`: Subnet, gateway, and IP range configuration
- `internal`: Creates an internal network with no external connectivity
- `ipv6`: Enables IPv6 networking
- `labels`: Custom metadata labels

### Unsupported Network Options

Some Docker Compose network options are not supported by Podman quadlet:

- `attachable`: Not applicable to Podman's network model
- `name`: Podman uses the network unit name
- `DNSEnabled`: This option is not supported by podman-systemd