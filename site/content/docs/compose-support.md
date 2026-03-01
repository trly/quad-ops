---
title: "Compose Feature Support"
weight: 15
---

# Docker Compose Feature Support

Quad-Ops converts [Docker Compose](https://compose-spec.io/) version 3.x+ configurations into
systemd-managed containers through [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html).
The following annotated compose file shows which [Docker Compose](https://compose-spec.io/) features are supported and how they map to
[Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) directives.

## Supported Features

```yaml
# compose.yaml — quad-ops supported features reference
# Comments show the Quadlet directive each option maps to

name: myapp                                    # project name (optional; directory name used if absent)

services:
  web:
    image: nginx:1.25                          # → Image (required)
    container_name: myapp-web                  # → ContainerName
    hostname: webhost                          # → HostName
    domainname: example.local                  # → HostName (overrides hostname if both set)
    command: ["nginx", "-g", "daemon off;"]    # → Exec
    entrypoint: ["/docker-entrypoint.sh"]      # → Entrypoint
    working_dir: /app                          # → WorkingDir

    # Networking
    ports:
      - "8080:80/tcp"                          # → PublishPort
      - "127.0.0.1:8443:443/tcp"
    expose:
      - "9090"                                 # → ExposeHostPort
    networks:                                  # → Network (or use network_mode: host | bridge)
      - frontend
      - backend
    dns: ["8.8.8.8", "1.1.1.1"]               # → DNS
    dns_search: [example.com]                  # → DNSSearch
    dns_opt: [ndots:1]                         # → DNSOption
    extra_hosts:
      api.internal: ["10.0.0.5"]               # → AddHost

    # Environment
    environment:
      APP_ENV: production                      # → Environment
      LOG_LEVEL: info
    env_file:
      - ./common.env                           # → EnvironmentFile
    labels:
      app: myapp                               # → Label.app
      version: "1.0"                           # → Label.version

    # Storage
    volumes:
      - data:/var/lib/nginx/data               # named volume → Volume
      - /host/config:/etc/nginx/conf.d:ro      # bind mount → Volume
    devices:
      - source: /dev/dri                       # → AddDevice
        target: /dev/dri
    read_only: true                            # → ReadOnly

    # Security
    privileged: true                            # → PodmanArgs --privileged
    cap_add: [NET_ADMIN]                       # → AddCapability
    cap_drop: [ALL]                            # → DropCapability
    group_add: ["wheel"]                       # → Group
    ipc: private                               # private or shareable only → Ipc
    pid: host                                  # → Pid
    security_opt:
      - label=disable                          # → SecurityLabelDisable
      - label=nested                           # → SecurityLabelNested
      - label=type:container_t                 # → SecurityLabelType
      - label=level:s0                         # → SecurityLabelLevel
      - label=filetype:container_file_t        # → SecurityLabelFileType
      - no-new-privileges                      # → NoNewPrivileges
      - seccomp=/etc/seccomp.json              # → SeccompProfile
      - mask=/proc/kcore                       # → Mask
      - unmask=/proc/self                      # → Unmask

    # Resources
    mem_limit: 512m                            # → Memory
    memswap_limit: 1g                          # → MemorySwap
    mem_reservation: 256m                      # → MemoryReservation
    cpus: 0.5                                  # → Cpus
    cpu_shares: 1024                           # → CpuWeight
    cpuset: "0,1"                              # → CpuSet
    pids_limit: 1024                           # → PidsLimit
    shm_size: 64m                              # → ShmSize
    oom_score_adj: -500                        # → OomScoreAdj
    # oom_kill_disable: true                   # → OomScoreAdj=-999 (alternative to oom_score_adj)
    sysctls:
      net.ipv4.ip_forward: "1"                 # → Sysctl.net.ipv4.ip_forward
    ulimits:
      nofile:
        soft: 1024                             # → Ulimit.nofile
        hard: 2048

    # Lifecycle
    restart: unless-stopped                    # no | always | on-failure | unless-stopped
    stop_signal: SIGTERM                       # SIGTERM | SIGKILL | TERM | KILL
    stop_grace_period: 30s                     # → StopTimeout
    pull_policy: always                        # → Pull
    init: true                                 # → RunInit
    tty: true                                  # → Tty
    stdin_open: true                           # → Interactive

    # Health check
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost/ || exit 1"]
      interval: 30s                            # → HealthInterval
      timeout: 10s                             # → HealthTimeout
      retries: 3                               # → HealthRetries
      start_period: 15s                        # → HealthStartPeriod
      start_interval: 5s                       # → HealthStartupInterval

    # Logging
    logging:
      driver: journald                         # json-file | journald
      options:
        tag: myapp-web                         # → LogOpt.tag

    # Dependencies
    depends_on:
      db:
        condition: service_started             # only supported condition

    # quad-ops extensions
    x-quad-ops-env-secrets:
      db-password-secret: DATABASE_PASSWORD    # → Secret=name,type=env,target=VAR
    x-quad-ops-annotations:
      io.podman.annotations.app: myapp         # → Annotation
    x-quad-ops-mounts:
      - "type=tmpfs,destination=/run,mode=1777" # → Mount
    x-quad-ops-podman-args:
      - "--log-level=warn"                     # → PodmanArgs
    x-quad-ops-container-args:
      - "--timeout=300"                        # → PodmanArgs (container-specific)

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: "${DB_PASSWORD}"       # variable interpolation from .env
    volumes:
      - db-data:/var/lib/postgresql/data
    restart: always

volumes:
  data:
    driver: local                              # only 'local' driver supported
    driver_opts:
      device: /dev/sda1                        # → Device
      type: ext4                               # → Type
      o: rw,relatime                           # → Options
      copy: "true"                             # → Copy
      user: "1000"                             # → User
      group: "1000"                            # → Group
    labels:
      managed-by: quad-ops                     # → Label.managed-by
    # x-quad-ops-podman-args: [...]            # → GlobalArgs
    # x-quad-ops-volume-args: [...]            # → PodmanArgs
  db-data:
    driver: local
    external: false                            # external: true = reference existing volume

networks:
  frontend:
    driver: bridge                             # only 'bridge' driver supported
    driver_opts:
      internal: "true"                         # → Internal
      ipv6: "true"                             # → IPv6
      disable_dns: "true"                      # → DisableDNS
      dns: "192.168.55.1"                      # → DNS
      interface_name: enp1                     # → InterfaceName
      ipam_driver: dhcp                        # → IPAMDriver
      ip_range: 172.20.0.128/25               # → IPRange
      network_delete_on_stop: "true"           # → NetworkDeleteOnStop
    ipam:
      config:
        - subnet: 172.20.0.0/16               # → Subnet
          gateway: 172.20.0.1                  # → Gateway
          ip_range: 172.20.0.0/24              # → IPRange
    labels:
      managed-by: quad-ops                     # → Label.managed-by
    # x-quad-ops-podman-args: [...]            # → PodmanArgs
    # x-quad-ops-network-args: [...]           # → PodmanArgs
  backend:
    driver: bridge
    external: false                            # external: true = reference existing network
```

## Unsupported Features

The following features will produce quadlet compatibility errors during validation.
Many of these can be worked around using `x-quad-ops-podman-args` to pass the
equivalent podman flag directly — see the [Bypassing Validation](#bypassing-validation)
section below.

```yaml
services:
  bad:
    image: nginx
    user: "nobody"                             # rejected — use x-quad-ops-podman-args: ["--user=nobody"]
    tmpfs: [/tmp]                              # rejected — use x-quad-ops-mounts or x-quad-ops-podman-args: ["--tmpfs=/tmp"]
    profiles: [debug]                          # rejected — no podman equivalent
    network_mode: none                         # rejected — use x-quad-ops-podman-args: ["--network=none"]
    network_mode: container:other              # rejected — use x-quad-ops-podman-args: ["--network=container:other"]
    network_mode: host
    ports: ["8080:80"]                         # rejected when using host network mode
    ipc: host                                  # rejected — use x-quad-ops-podman-args: ["--ipc=host"]
    ipc: service:other                         # rejected — use x-quad-ops-podman-args: ["--ipc=service:other"]
    ipc: container:other                       # rejected — use x-quad-ops-podman-args: ["--ipc=container:other"]
    security_opt:
      - apparmor=unconfined                    # rejected — use x-quad-ops-podman-args: ["--security-opt=apparmor=unconfined"]
    stop_signal: SIGINT                        # rejected — use x-quad-ops-podman-args: ["--stop-signal=SIGINT"]
    logging:
      driver: splunk                           # rejected — use x-quad-ops-podman-args: ["--log-driver=splunk"]
    depends_on:
      db:
        condition: service_healthy             # rejected (only service_started) — no podman equivalent
        condition: service_completed_successfully # rejected — no podman equivalent
    deploy:
      replicas: 3                              # rejected — one instance per systemd unit, no workaround
      placement:
        constraints: ["node.role==manager"]    # rejected — Swarm feature, no workaround
        preferences:
          - spread: datacenter                 # rejected — Swarm feature, no workaround

volumes:
  bad-vol:
    driver: nfs                                # rejected — use x-quad-ops-volume-args to pass driver options

networks:
  bad-net:
    driver: overlay                            # rejected — use x-quad-ops-network-args to pass driver options
```

### Bypassing Validation

The `x-quad-ops-podman-args` and `x-quad-ops-container-args` extensions pass arbitrary
flags directly to `podman run` via the Quadlet `PodmanArgs=` directive. This is an
intentional escape hatch: remove the rejected compose key and use the equivalent podman
flag instead.

For example, to use `tmpfs` mounts (rejected as a compose key):

```yaml
services:
  web:
    image: nginx:latest
    # tmpfs: [/tmp]                            # ← would be rejected
    x-quad-ops-podman-args:
      - "--tmpfs=/tmp"                         # ← passes directly to podman
```

Or use `x-quad-ops-mounts` for more control:

```yaml
services:
  web:
    image: nginx:latest
    x-quad-ops-mounts:
      - "type=tmpfs,destination=/tmp,tmpfs-size=100m"
```

Similarly for volume and network extensions:
- `x-quad-ops-volume-args` passes flags to `podman volume create`
- `x-quad-ops-network-args` passes flags to `podman network create`

> **Note:** No validation is performed on values passed through these extensions.
> Ensure the flags are valid for your version of Podman.

## quad-ops Extensions

Quad-Ops provides custom compose extensions (prefixed with `x-quad-ops-`) that map to
Podman Quadlet directives not directly expressible through standard Docker Compose syntax.

### Service Extensions

#### `x-quad-ops-env-secrets`

Maps [Podman secrets](https://docs.podman.io/en/latest/markdown/podman-secret.1.html) to environment variables inside the container. Requires Podman 4.5+.

Each entry maps a secret name (which must already exist in Podman via `podman secret create`) to the environment variable it should be exposed as.

**Quadlet directive:** `Secret=<name>,type=env,target=<VAR>`

```yaml
services:
  web:
    image: myapp:latest
    x-quad-ops-env-secrets:
      db-password-secret: DATABASE_PASSWORD
      api-key-secret: API_KEY
```

Generated output:

```ini
[Container]
Secret=api-key-secret,type=env,target=API_KEY
Secret=db-password-secret,type=env,target=DATABASE_PASSWORD
```

#### `x-quad-ops-annotations`

Adds [OCI annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md) to the container. These are distinct from labels — annotations are metadata attached to the container runtime rather than the container image.

**Quadlet directive:** `Annotation=<key>=<value>`

```yaml
services:
  web:
    image: myapp:latest
    x-quad-ops-annotations:
      io.podman.annotations.app: myapp
      io.podman.annotations.version: "1.0"
```

Generated output:

```ini
[Container]
Annotation=io.podman.annotations.app=myapp
Annotation=io.podman.annotations.version=1.0
```

#### `x-quad-ops-mounts`

Specifies advanced mount options using Podman's `--mount` flag syntax. Use this for mount types not expressible through the standard `volumes` key, such as tmpfs mounts with specific options.

**Quadlet directive:** `Mount=<mount-spec>`

```yaml
services:
  web:
    image: myapp:latest
    x-quad-ops-mounts:
      - "type=tmpfs,destination=/run,mode=1777"
      - "type=tmpfs,destination=/tmp,tmpfs-size=100m"
```

Generated output:

```ini
[Container]
Mount=type=tmpfs,destination=/run,mode=1777
Mount=type=tmpfs,destination=/tmp,tmpfs-size=100m
```

#### `x-quad-ops-podman-args` (service)

Passes additional arguments to the `podman` command when running the container. Use this for Podman features that have no equivalent in Docker Compose or Quadlet directives.

**Quadlet directive:** `PodmanArgs=<arg>`

```yaml
services:
  web:
    image: myapp:latest
    x-quad-ops-podman-args:
      - "--log-level=warn"
      - "--sdnotify=conmon"
```

Generated output:

```ini
[Container]
PodmanArgs=--log-level=warn
PodmanArgs=--sdnotify=conmon
```

#### `x-quad-ops-container-args`

Passes container-specific arguments to Podman. Functionally identical to `x-quad-ops-podman-args` on services — both generate `PodmanArgs=` directives. Use whichever name better communicates intent.

**Quadlet directive:** `PodmanArgs=<arg>`

```yaml
services:
  web:
    image: myapp:latest
    x-quad-ops-container-args:
      - "--timeout=300"
```

Generated output:

```ini
[Container]
PodmanArgs=--timeout=300
```

### Volume Extensions

#### `x-quad-ops-podman-args` (volume)

Passes global arguments to the `podman volume create` command.

**Quadlet directive:** `GlobalArgs=<arg>`

```yaml
volumes:
  data:
    driver: local
    x-quad-ops-podman-args:
      - "--log-level=debug"
```

Generated output:

```ini
[Volume]
GlobalArgs.0=--log-level=debug
```

#### `x-quad-ops-volume-args`

Passes volume-specific arguments to the `podman volume create` command.

**Quadlet directive:** `PodmanArgs=<arg>`

```yaml
volumes:
  data:
    driver: local
    x-quad-ops-volume-args:
      - "--opt=type=nfs"
```

Generated output:

```ini
[Volume]
PodmanArgs.0=--opt=type=nfs
```

### Network Extensions

#### `x-quad-ops-podman-args` (network)

Passes additional arguments to the `podman network create` command.

**Quadlet directive:** `PodmanArgs=<arg>`

```yaml
networks:
  frontend:
    driver: bridge
    x-quad-ops-podman-args:
      - "--opt=mtu=9000"
```

Generated output:

```ini
[Network]
PodmanArgs=--opt=mtu=9000
```

#### `x-quad-ops-network-args`

Passes network-specific arguments to the `podman network create` command. Functionally identical to `x-quad-ops-podman-args` on networks — both generate `PodmanArgs=` directives.

**Quadlet directive:** `PodmanArgs=<arg>`

```yaml
networks:
  frontend:
    driver: bridge
    x-quad-ops-network-args:
      - "--disable-dns"
```

Generated output:

```ini
[Network]
PodmanArgs=--disable-dns
```
