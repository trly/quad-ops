---
title: "Resource Constraints & Advanced Config"
weight: 50
---

# Resource Constraints & Advanced Configuration

Quad-Ops supports Docker Compose resource constraints and advanced container configuration, mapping them to Podman Quadlet directives or implementing them via PodmanArgs.

## Resource Constraints

Resource constraints allow you to limit and reserve system resources for containers.

### Resource Constraints Support

Quad-Ops supports resource constraints from Docker Compose files by using Podman Quadlet's PodmanArgs directive. While these features aren't directly supported by Podman Quadlet, Quad-Ops automatically converts them to equivalent podman run arguments.

| Docker Compose Property | Supported in Compose | Implementation Method | Description |
|--------------------------|---------------------|---------------------|-------------|
| `mem_limit` | ✅ | PodmanArgs | Memory limit (e.g., "512m", "2g") |
| `mem_reservation` | ✅ | PodmanArgs | Soft memory limit/reservation |
| `memswap_limit` | ✅ | PodmanArgs | Memory plus swap limit |
| `cpu_shares` | ✅ | PodmanArgs | CPU shares (relative weight) |
| `cpu_quota` | ✅ | PodmanArgs | Limit CPU CFS quota |
| `cpu_period` | ✅ | PodmanArgs | Limit CPU CFS period |
| `pids_limit` | ✅ | Native | Limit number of processes |

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

| Docker Compose Property | Implementation | Description |
|--------------------------|--------------------------|-------------|
| `ulimits` | Native (`Ulimit`) | Resource limits (file descriptors, processes) |
| `sysctls` | Native (`Sysctl`) | Kernel parameters |
| `tmpfs` | Native (`Tmpfs`) | Mount tmpfs volumes |
| `userns_mode` | Native (`UserNS`) | User namespace mode |
| `cap_add` | PodmanArgs | Add Linux capabilities |
| `cap_drop` | PodmanArgs | Drop Linux capabilities |
| `devices` | PodmanArgs | Device mappings into container |
| `dns` | PodmanArgs | Custom DNS servers |
| `dns_search` | PodmanArgs | Custom DNS search domains |
| `dns_opt` | PodmanArgs | DNS options |
| `ipc` | PodmanArgs | IPC namespace mode |
| `pid` | PodmanArgs | PID namespace mode |
| `shm_size` | PodmanArgs | Size of /dev/shm |
| `mac_address` | PodmanArgs | Container MAC address |
| `cgroup_parent` | PodmanArgs | Parent cgroup for container |
| `runtime` | PodmanArgs | Custom container runtime |
| `storage_opt` | PodmanArgs | Storage driver options |
| `security_opt` | PodmanArgs | Security options & labels |
| `privileged` | PodmanArgs | Run container in privileged mode |

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
    # Features implemented via PodmanArgs
    cap_add:
      - SYS_PTRACE
    devices:
      - "/dev/sda:/dev/xvda:rwm"
    dns:
      - 8.8.8.8
      - 1.1.1.1
```

## Implementation Details

Quad-Ops reads and processes Docker Compose resource constraints and advanced configuration features using two approaches:

1. **Native Support**: Features directly supported by Podman Quadlet are converted to their corresponding Quadlet directives.
2. **PodmanArgs Support**: Features not directly supported by Podman Quadlet are implemented using the `PodmanArgs` directive, which passes arguments directly to the `podman run` command.

### How PodmanArgs Works

PodmanArgs is a special directive in Podman Quadlet that allows passing arguments directly to the podman run command. When Quad-Ops encounters a Docker Compose feature not natively supported by Quadlet, it automatically adds the appropriate podman run argument via the PodmanArgs directive.

For example, when processing `mem_limit: 512m`, Quad-Ops generates:
```
[Container]
PodmanArgs=--memory=512m
```

You will see informational messages during conversion like: 
```
Service 'api' uses Memory limits (mem_limit) which is not directly supported by Podman Quadlet. Using PodmanArgs directive instead.
```

This approach ensures that all Docker Compose features are properly implemented, even when Podman Quadlet doesn't have direct support for them.

### Notes

- Resource constraints can be specified using the deploy section or directly on service level
- Memory values can use suffixes (b, k, m, g) or be specified in bytes
- Quad-Ops automatically handles all format conversions (like float to int or byte count conversions)
- All services with unsupported features will generate clear warnings that indicate how they're being handled