# Sysctls Example

This example demonstrates how to use Docker Compose `sysctls` field for kernel parameter tuning in quad-ops.

## Overview

The `sysctls` field allows you to set kernel parameters for containers. This is commonly used for:

- **Network tuning**: Configure IP forwarding, TCP keepalive, connection limits
- **Shared memory**: Adjust kernel shared memory limits for databases
- **Performance optimization**: Fine-tune kernel parameters for specific workloads

## Services

### Router Service

Uses common network sysctls:

```yaml
sysctls:
  net.ipv4.ip_forward: "1"        # Enable IP forwarding for routing
  net.core.somaxconn: "1024"      # Maximum queued connections
```

### Database Service

Uses kernel shared memory parameters for PostgreSQL:

```yaml
sysctls:
  kernel.shmmax: "68719476736"           # Max shared memory segment (64GB)
  kernel.shmall: "4294967296"            # Total shared memory (4GB pages)
  net.ipv4.tcp_keepalive_time: "600"     # TCP keepalive timer (10 min)
  net.ipv4.tcp_keepalive_intvl: "60"     # TCP keepalive interval (60s)
```

## Rendering

### systemd (Quadlet)

On Linux with systemd, sysctls are rendered using the native Quadlet `Sysctl=` directive:

```ini
[Container]
Image=nginx:alpine
Sysctl=net.core.somaxconn=1024
Sysctl=net.ipv4.ip_forward=1
```

### launchd (macOS)

On macOS, sysctls are rendered as Podman command-line flags:

```bash
podman run --rm \
  --sysctl net.ipv4.ip_forward=1 \
  --sysctl net.core.somaxconn=1024 \
  nginx:alpine
```

## Usage

```bash
# Synchronize (dry-run)
quad-ops sync --dry-run --user

# Synchronize (apply changes)
quad-ops sync --user

# Check status
quad-ops up --user

# Stop services
quad-ops down --user
```

## Security Considerations

- Sysctls require elevated privileges or specific capabilities
- Only use sysctls you understand and trust
- Some sysctls may require `--privileged` or `CAP_SYS_ADMIN`
- Test sysctls in development before production use

## Common Use Cases

**IP Forwarding (Routing)**:
```yaml
sysctls:
  net.ipv4.ip_forward: "1"
```

**Database Shared Memory**:
```yaml
sysctls:
  kernel.shmmax: "68719476736"
  kernel.shmall: "4294967296"
```

**High-Performance Networking**:
```yaml
sysctls:
  net.core.somaxconn: "4096"
  net.ipv4.tcp_max_syn_backlog: "8192"
  net.ipv4.tcp_tw_reuse: "1"
```

**Debugging**:
```yaml
sysctls:
  kernel.core_pattern: "/tmp/core.%e.%p"
```

## References

- [Podman systemd.unit(5)](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- [Docker Compose sysctls](https://docs.docker.com/compose/compose-file/05-services/#sysctls)
- [Podman run --sysctl](https://docs.podman.io/en/latest/markdown/podman-run.1.html)
- [Linux kernel parameters](https://www.kernel.org/doc/html/latest/admin-guide/sysctl/index.html)
