---
title: "Docker Compose Support"
weight: 20
---

# Docker Compose Support

Quad-Ops provides comprehensive support for Docker Compose version 3.x files, converting them to Podman Quadlet units for systemd management.

## Supported Compose Versions

- **No version specified** (treated as 3.x) **[Recommended]**
- **Version 3.0** through **3.8** (latest)
- **Version 2.x** (partial compatibility)

## Core Service Configuration

### Container Basics

```yaml
services:
  web:
    image: docker.io/library/nginx:latest
    container_name: custom-web  # Optional custom naming
    hostname: web-server        # Internal hostname
    restart: unless-stopped     # Restart policy
    command: ["nginx", "-g", "daemon off;"]
    working_dir: /app
    user: "1000:1000"
    labels:
      - "app=web"
      - "version=1.0"
```

### Port Configuration

```yaml
services:
  app:
    ports:
      - "8080:80"           # Host:container
      - "443:443"           # HTTPS
      - "127.0.0.1:3000:3000" # Bind to specific interface
      - "9000"              # Random host port
```

### Volume Mounts

```yaml
services:
  app:
    volumes:
      - "./data:/app/data"              # Bind mount
      - "logs:/var/log"                 # Named volume
      - "/etc/localtime:/etc/localtime:ro" # Read-only mount
      - "cache:/tmp:Z"                  # SELinux label

volumes:
  logs:
    driver: local
  cache:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
```

### Environment Variables

```yaml
services:
  app:
    environment:
      - NODE_ENV=production
      - DEBUG=false
      - DATABASE_URL=postgresql://user:pass@db:5432/app
    env_file:
      - .env
      - .env.production
```

## Network Configuration

### Custom Networks

```yaml
services:
  web:
    networks:
      - frontend
      - backend

  db:
    networks:
      - backend

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No external access
```

### Network Aliases

```yaml
services:
  app:
    networks:
      backend:
        aliases:
          - api
          - api-server

  database:
    networks:
      backend:
        aliases:
          - db
          - postgres
```

## Service Dependencies

### Basic Dependencies

```yaml
services:
  web:
    depends_on:
      - db
      - redis

  db:
    image: postgres:13

  redis:
    image: redis:alpine
```

### Advanced Dependencies (Docker Compose 3.8+)

```yaml
services:
  web:
    depends_on:
      db:
        condition: service_healthy

  db:
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "postgres"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

## Resource Constraints

### Memory Limits

```yaml
services:
  app:
    mem_limit: 512m
    mem_reservation: 256m
    memswap_limit: 1g

    # Or using deploy section
    deploy:
      resources:
        limits:
          memory: 512m
        reservations:
          memory: 256m
```

### CPU Limits

```yaml
services:
  app:
    cpus: "1.5"           # 1.5 CPU cores
    cpu_percent: 50       # 50% of available CPU
    cpu_shares: 1024      # Relative weight
    cpu_quota: "150000"   # CPU quota in microseconds
    cpu_period: "100000"  # CPU period in microseconds

    # Or using deploy section
    deploy:
      resources:
        limits:
          cpus: "1.5"
```

### Process Limits

```yaml
services:
  app:
    pids_limit: 100  # Maximum number of processes
```

## Health Checks

```yaml
services:
  web:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s      # Check every 30 seconds
      timeout: 10s       # 10 second timeout
      retries: 3         # Retry 3 times before unhealthy
      start_period: 40s  # Wait 40s before first check
      start_interval: 5s # Check every 5s during start period

  # Disable health check
  app-no-health:
    healthcheck:
      disable: true
```

## Build Configuration

### Basic Build

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.prod
      args:
        - NODE_ENV=production
        - VERSION=1.0.0
      target: production
```

### Advanced Build

```yaml
services:
  app:
    build:
      context: .
      dockerfile: multi-stage.Dockerfile
      target: production
      args:
        NODE_ENV: production
        VERSION: "1.0.0"
      labels:
        - "app=myapp"
        - "version=1.0.0"
      network: host
      pull: true
      secrets:
        - source: api_key
          target: /run/secrets/api_key
```

## Security Configuration

### Privileged Containers

```yaml
services:
  system-app:
    privileged: true
    cap_add:
      - SYS_ADMIN
      - NET_ADMIN
    cap_drop:
      - SETUID
      - SETGID
```

### Security Labels

```yaml
services:
  app:
    security_opt:
      - "label=disable"
      - "no-new-privileges:true"
```

### User Namespaces

```yaml
services:
  app:
    user: "1000:1000"
    userns_mode: "host"
```

## Advanced Features

### Device Access

```yaml
services:
  hardware-app:
    devices:
      - "/dev/sda:/dev/sda"
      - "/dev/ttyUSB0:/dev/ttyUSB0"
    device_cgroup_rules:
      - "c 1:3 rmw"  # Allow read/write to /dev/null
```

### Shared Memory

```yaml
services:
  app:
    shm_size: "2gb"
    ipc: "shareable"  # or "container:other-container"
```

### Network Configuration

```yaml
services:
  app:
    network_mode: "host"  # or "bridge", "none"
    dns:
      - 8.8.8.8
      - 1.1.1.1
    dns_search:
      - example.com
    extra_hosts:
      - "api:192.168.1.100"
      - "db:192.168.1.101"
    mac_address: "02:42:ac:11:00:02"
```

### Logging Configuration

```yaml
services:
  app:
    logging:
      driver: journald
      options:
        tag: "myapp"
        labels: "service"
```

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

### Volume Extensions

Podman-specific volume options:

```yaml
services:
  app:
    volumes:
      - "data:/data"
    x-podman-volumes:
      - "cache:/tmp/cache:O"  # Overlay mount
      - "logs:/logs:U"        # Chown to container user
```

### Build Extensions

Additional build arguments:

```yaml
services:
  app:
    build:
      context: .
    x-podman-buildargs:
      BUILDKIT_INLINE_CACHE: "1"
      BUILDPLATFORM: "linux/amd64"
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

## Best Practices

### Image Naming

Always use fully qualified image names:

```yaml
# ✅ Good
services:
  web:
    image: docker.io/library/nginx:latest

# ❌ Avoid
services:
  web:
    image: nginx  # May cause registry resolution issues
```

### Volume Paths

Ensure bind mount directories exist:

```yaml
services:
  app:
    volumes:
      - "./data:/app/data"  # Ensure ./data exists
      - "/host/logs:/logs"  # Ensure /host/logs exists
```

### Network Design

Use custom networks for service isolation:

```yaml
# ✅ Good - Explicit network design
services:
  frontend:
    networks: [web]
  backend:
    networks: [web, data]
  database:
    networks: [data]

networks:
  web:
  data:
    internal: true

# ❌ Avoid - Default network for everything
services:
  frontend:
  backend:
  database:
```

### Dependency Management

Use `depends_on` for startup ordering:

```yaml
services:
  app:
    depends_on: [db, redis]  # Start db and redis first
  db:
  redis:
```

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| **Image pull failed** | Unqualified image name | Add registry prefix (docker.io/) |
| **Volume mount failed** | Directory doesn't exist | Create bind mount directories |
| **Network not found** | Missing network definition | Add to networks section |
| **Service won't start** | Missing dependencies | Check depends_on relationships |

### Validation Commands

```bash
# Validate compose file syntax
docker-compose -f docker-compose.yml config

# Test compose conversion
quad-ops sync --dry-run

# Check generated units
ls /etc/containers/systemd/
```

## Next Steps

- [Environment Files](environment-files) - Environment variable management
- [Build Support](build-support) - Docker build configurations
- [Supported Features](../podman-systemd/supported-features) - Feature compatibility matrix