---
title: "Resource Constraints & Advanced Config"
weight: 50
---

# Resource Constraints & Advanced Configuration

Quad-Ops supports Docker Compose resource constraints and advanced container configuration, mapping them to Podman Quadlet directives.

## Resource Constraints

Resource constraints allow you to limit and reserve system resources for containers.

### Resource Constraints Support

Quad-Ops tracks resource constraints from Docker Compose files internally but has limited support for generating Podman Quadlet properties due to Quadlet's capabilities.

| Docker Compose Property | Supported in Compose | Supported in Quadlet | Description |
|--------------------------|---------------------|---------------------|-------------|
| `mem_limit` | ✅ | ❌ | Memory limit (e.g., "512m", "2g") |
| `mem_reservation` | ✅ | ❌ | Soft memory limit/reservation |
| `memswap_limit` | ✅ | ❌ | Memory plus swap limit |
| `cpu_shares` | ✅ | ❌ | CPU shares (relative weight) |
| `cpu_quota` | ✅ | ❌ | Limit CPU CFS quota |
| `cpu_period` | ✅ | ❌ | Limit CPU CFS period |
| `pids_limit` | ✅ | ✅ | Limit number of processes |

### Example

```yaml
services:
  api:
    image: docker.io/myapp:latest
    mem_limit: 512m
    mem_reservation: 256m
    cpu_shares: 512
    cpu_quota: 50000
    cpu_period: 100000
    pids_limit: 100
```

## Advanced Container Configuration

Advanced configuration provides more granular control over container behavior.

### Supported Advanced Configuration

| Docker Compose Property | Podman Quadlet Property | Description |
|--------------------------|--------------------------|-------------|
| `ulimits` | `Ulimit` | Resource limits (file descriptors, processes) |
| `sysctls` | `Sysctl` | Kernel parameters |
| `tmpfs` | `Tmpfs` | Mount tmpfs volumes |
| `userns_mode` | `UserNS` | User namespace mode |

### Example

```yaml
services:
  web:
    image: docker.io/nginx:latest
    ulimits:
      nofile:
        soft: 20000
        hard: 40000
      nproc: 65535
    sysctls:
      net.core.somaxconn: "1024"
      net.ipv4.ip_forward: "1"
    tmpfs:
      - /tmp
      - /run:rw,size=1G
    userns_mode: keep-id
```

## Implementation Details

Quad-Ops reads and processes Docker Compose resource constraints and maps supported configuration to Podman Quadlet directives during the conversion process. These directives are included in the generated container unit files that systemd uses to manage the containers.

### Notes

- Resource constraints can be specified using the deploy section or directly on service level
- Memory values can use suffixes (b, k, m, g) or be specified in bytes
- Some advanced features like device_cgroup_rules are not yet implemented
- For security options where Podman Quadlet doesn't match Docker's exactly, we map to the closest equivalent

### Limitations

- Most memory and CPU constraints defined in Docker Compose files **cannot** be represented in Podman Quadlet files
- Quad-Ops will track these values internally and log warnings when it encounters unsupported features
- You will see warning messages like: `Service 'api' uses Memory limits (mem_limit) which is not supported by Podman Quadlet. This setting will be ignored.`
- Only process limits (`pids_limit`) are fully supported among resource constraints
- This is a limitation of Podman Quadlet, not Quad-Ops itself
- Despite these limitations, Quad-Ops will still properly convert and manage your services