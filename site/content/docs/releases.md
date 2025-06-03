---
title: "Release Notes"
weight: 60
---

# Release Notes

## Upcoming Features

### Init Containers Support

**NEW**: Support for `x-quad-ops-init` extension in Docker Compose files.

#### Overview

Similar to Kubernetes init containers, you can now define containers that run before your main service starts. This is useful for:

- Database migrations
- Configuration setup
- Service dependency checks
- File system preparation

#### Example

```yaml
services:
  web:
    image: nginx:alpine
    x-quad-ops-init:
      - image: busybox:latest
        command: ["sh", "-c", "echo 'Initializing...' && sleep 2"]
      - image: alpine:latest
        command: "mkdir -p /data && echo 'Ready' > /data/status"
```

#### Features

- **Multiple init containers** - Run sequentially in defined order
- **Automatic dependencies** - Main service waits for all init containers
- **Oneshot services** - Uses `Type=oneshot` with `RemainAfterExit=yes`
- **Error handling** - Main service won't start if init containers fail
- **Flexible commands** - String or array syntax support

#### Generated Units

Each init container creates a separate systemd unit:
- Naming: `<project>-<service>-init-<index>.container`
- Dependencies: Main service automatically depends on all init containers

#### Documentation

- [Init Containers Guide](container-management/init-containers) - Complete usage guide
- [Docker Compose Support](container-management/docker-compose-support) - Extension reference
- [Examples](https://github.com/trly/quad-ops/tree/main/examples/init-containers) - Working examples

---

*This changelog will be updated with each release.*
