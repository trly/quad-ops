---
title: "quad-ops"
weight: 0
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# ![quad-ops](images/quad-ops-64.png) quad-ops

## GitOps for Quadlet
![GitHub License](https://img.shields.io/github/license/trly/quad-ops)
![Docs Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml)
![Build Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/docs.yaml?label=docs)
![CodeQL Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml?label=codeql)
![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops)
[![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

A cross-platform GitOps framework for container management with native service integration

Quad-Ops is a tool that helps you manage container deployments using a GitOps workflow.
It watches Git repositories for standard [Docker Compose](https://compose-spec.io/) files and automatically converts them into native service definitions for your platform:

- **Linux**: systemd + [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- **macOS**: launchd (planned)

## Docker Compose Feature Support

Quad-Ops converts [Docker Compose](https://compose-spec.io/) version 3.x+ configurations into
systemd-managed containers through [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html).
The following annotated compose file shows which [Docker Compose](https://compose-spec.io/) features are supported and how they map to
[Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) directives.

### Supported Features

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
    x-podman-env-secrets:
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

### Unsupported Features

The following features will produce quadlet compatibility errors during validation:

```yaml
services:
  bad:
    image: nginx
    privileged: true                           # rejected
    user: "nobody"                             # rejected — use systemd user mapping
    tmpfs: [/tmp]                              # rejected — use volumes or x-quad-ops-mounts
    profiles: [debug]                          # rejected
    network_mode: none                         # rejected (only bridge or host)
    network_mode: container:other              # rejected
    network_mode: host
    ports: ["8080:80"]                         # rejected when using host network mode
    ipc: host                                  # rejected (only private or shareable)
    ipc: service:other                         # rejected
    ipc: container:other                       # rejected
    security_opt:
      - apparmor=unconfined                    # rejected
    stop_signal: SIGINT                        # rejected (only SIGTERM or SIGKILL)
    logging:
      driver: splunk                           # rejected (only json-file or journald)
    depends_on:
      db:
        condition: service_healthy             # rejected (only service_started)
        condition: service_completed_successfully # rejected
    deploy:
      replicas: 3                              # rejected — one instance per systemd unit
      placement:
        constraints: ["node.role==manager"]    # rejected — Swarm feature
        preferences:
          - spread: datacenter                 # rejected — Swarm feature

volumes:
  bad-vol:
    driver: nfs                                # rejected (only 'local')

networks:
  bad-net:
    driver: overlay                            # rejected (only 'bridge')
```
