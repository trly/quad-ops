---
title: "Docker Compose Support"
weight: 20
---

# Docker Compose Support

Quad-Ops converts Docker Compose files to Podman Quadlet units for systemd management. For comprehensive documentation on Docker Compose syntax and features, see the [Compose Specification](https://compose-spec.io/).

## Supported Compose Versions

- **No version specified** (treated as 3.x) **[Recommended]**
- **Version 3.0** through **3.8** (latest)
- **Version 2.x** (partial compatibility)

## Compose Specification Support

Quad-Ops supports all container runtime features that work with standalone Podman.

### Fully Supported

**Core container configuration:**
- `image`, `build`, `command`, `entrypoint`, `working_dir`, `user`, `hostname`

**Environment and labels:**
- `environment`, `env_file`, `labels`, `annotations`

**Networking:**
- `networks` (bridge, host, custom networks)
- `ports` (host mode only)
- `dns`, `dns_search`, `dns_opt`, `extra_hosts`
- `network_mode` (bridge, host, none, container:name)

**Storage:**
- `volumes` (bind mounts, named volumes, tmpfs)
- `secrets` with file/content/environment sources
- `configs` with file/content/environment sources

**Resources:**
- `memory`, `cpu_shares`, `cpu_quota`, `cpu_period`
- `pids_limit`, `shm_size`, `sysctls`, `ulimits`

**Security:**
- `cap_add`, `cap_drop`, `privileged`, `security_opt`, `read_only`
- `group_add`, `pid` mode, `ipc` mode, `cgroup_parent`

**Devices and hardware:**
- `devices`, `device_cgroup_rules`
- `runtime` (e.g., nvidia for GPU support)

**Health and lifecycle:**
- `healthcheck` (test, interval, timeout, retries, start_period)
- `restart` (maps to systemd restart policies)
- `stop_signal`, `stop_grace_period`
- `depends_on` (maps to systemd After/Requires)

### Partially Supported

**Secrets and configs:**
- File sources (`file: ./secret.txt`)
- Content sources (`content: "secret data"`)
- Environment sources (`environment: SECRET_VAR`)
- NOT supported: Swarm driver (`external: true` with `driver`)

**Resource constraints:**
- `deploy.resources.limits` (memory, cpus, pids)
- `deploy.resources.reservations` (partial - depends on cgroups v2)

**Dependency conditions:**
- All `depends_on` conditions (`service_started`, `service_healthy`, `service_completed_successfully`) map to systemd `After` + `Requires`
- No health-based startup gating (Quadlet limitation)

**Logging:**
- Supported: `journald`, `k8s-file`, `none`, `passthrough`
- NOT supported: Custom drivers not supported by Podman

### Not Supported - Use Alternatives

**Standard Compose fields:**
- `volumes_from` - Use named volumes or bind mounts
- `stdin_open`, `tty` - Interactive mode not practical in systemd units
- `extends` - Use YAML anchors or include directives

### Explicitly Out of Scope - Swarm Orchestration

Quad-Ops is **NOT** a Swarm orchestrator. These features are rejected with validation errors:

- `deploy.mode: global` - Multi-node replication
- `deploy.replicas > 1` - Multi-instance services
- `deploy.placement` - Node placement constraints
- `deploy.update_config`, `deploy.rollback_config` - Rolling updates
- `deploy.endpoint_mode` (vip/dnsrr) - Swarm service discovery
- `ports.mode: ingress` - Swarm load balancing (use `mode: host`)
- `configs`/`secrets` with `driver` field - Swarm secret store

**For these features, use:**
- **Kubernetes** - Cloud-native orchestration with full feature set
- **Nomad** - Lightweight orchestrator for VMs and containers
- **Docker Swarm** - If you need Swarm-specific features

**Reference:** [Podman Quadlet Documentation](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

## Podman-Specific Extensions

### Environment Secrets

Map Podman secrets to environment variables:

```yaml
services:
  app:
    environment:
      - DB_PASSWORD_FILE=/run/secrets/db_password
    x-podman-env-secrets:
      DB_PASSWORD: db_password  # secret name -> env var
      API_KEY: api_secret
```

## Conversion Examples

### Docker Compose to Quadlet

**Docker Compose:**
```yaml
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html
    depends_on:
      - app

  app:
    build: .
    environment:
      - NODE_ENV=production
```

**Generated Quadlet Units:**

`myproject-web.container`:
```ini
[Unit]
Description=myproject-web container
After=myproject-app.service

[Container]
Image=docker.io/library/nginx:latest
PublishPort=8080:80
Volume=./html:/usr/share/nginx/html
NetworkAlias=web

[Service]
Restart=always

[Install]
WantedBy=default.target
```

`myproject-app.container`:
```ini
[Unit]
Description=myproject-app container

[Container]
Image=localhost/myproject-app:latest
Environment=NODE_ENV=production
NetworkAlias=app

[Service]
Restart=always

[Install]
WantedBy=default.target
```

## Quad-Ops Validation

Validate Docker Compose files before deployment:

```bash
# Validate compose files with quad-ops extensions
quad-ops validate docker-compose.yml

# Validate all compose files in directory
quad-ops validate ./compose-files/

# Validate remote repository
quad-ops validate --repo https://github.com/user/repo.git

# Test compose conversion without applying
quad-ops sync --dry-run

# Check generated Quadlet units
ls /etc/containers/systemd/
```

The `validate` command checks for:
- Docker Compose syntax and structure
- Quad-ops extension compatibility  
- Security requirements (secrets, env vars)
- DNS naming conventions
- File path security

## Next Steps

- [Environment Files](environment-files) - Environment variable management
- [Build Support](build-support) - Docker build configurations
- [Supported Features](../podman-systemd/supported-features) - Feature compatibility matrix
- [Compose Specification](https://compose-spec.io/) - Official Docker Compose documentation