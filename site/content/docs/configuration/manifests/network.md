---
title: "Network"
weight: 30
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Network

## Options

> https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#network-units-network

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `label` | []string | - | Network labels (managed-by=quad-ops is added automatically) |
| `driver` | string | - | Network driver |
| `gateway` | string | - | Gateway address |
| `ip_range` | string | - | IP address range |
| `subnet` | string | - | Subnet CIDR |
| `ipv6` | bool | false | Enable IPv6 |
| `internal` | bool | false | Restrict external access |
| `dns_enabled` | bool | false | Enable DNS |
| `options` | []string | - | Additional network options |

## Example

```yaml
---
name: app-net
type: network
systemd:
  description: "Application network"
network:
  driver: "bridge"
  subnet: "172.20.0.0/16"
  gateway: "172.20.0.1"
  ipv6: true
  dns_enabled: true
```