---
title: "Init Containers"
weight: 25
---

# Init Containers

Init containers run before your main application containers start, similar to Kubernetes init containers. They're useful for setup tasks, database migrations, configuration preparation, or any initialization work that needs to complete before your main application runs.

## Overview

The `x-quad-ops-init` extension allows you to define one or more containers that run sequentially before the main service starts. If any init container fails, the main service will not start.

## Basic Syntax

```yaml
services:
  myservice:
    image: myapp:latest
    x-quad-ops-init:
      - image: busybox:latest
        command: "echo 'Preparing environment...'"
      - image: alpine:latest
        command: ["sh", "-c", "mkdir -p /data && touch /data/ready"]
```

## Configuration

### Required Fields

- **`image`** - The container image to run for initialization

### Optional Fields

- **`command`** - Command to execute in the init container
  - Can be a string: `"echo hello"`
  - Or an array: `["sh", "-c", "echo hello"]`
  - If omitted, uses the image's default command

## How It Works

### Unit Generation

Each init container generates a separate Quadlet unit:

- **Naming pattern**: `<project>-<service>-init-<index>`
- **Service type**: `oneshot` with `RemainAfterExit=yes`
- **Execution order**: Sequential in the order defined

### Dependency Management

The main service automatically depends on all init containers:

```ini
[Unit]
After=myproject-myservice-init-0.service myproject-myservice-init-1.service
Requires=myproject-myservice-init-0.service myproject-myservice-init-1.service
```

### Execution Flow

1. Init containers run **sequentially** in defined order
2. Each init container must complete successfully
3. Main service starts only after **all** init containers succeed
4. If any init container fails, the main service won't start

## Common Use Cases

### Database Migration

```yaml
services:
  web:
    image: myapp:latest
    depends_on:
      - database
    x-quad-ops-init:
      - image: migrate/migrate
        command: ["-path", "/migrations", "-database", "postgres://...", "up"]
  
  database:
    image: postgres:15
    environment:
      POSTGRES_DB: myapp
```

### File System Preparation

```yaml
services:
  app:
    image: nginx:alpine
    x-quad-ops-init:
      - image: busybox:latest
        command: ["sh", "-c", "mkdir -p /data/cache /data/logs && chmod 755 /data/*"]
      - image: alpine:latest
        command: "wget -O /data/config.json https://config-server/app-config"
    volumes:
      - app-data:/data
```

### Service Dependencies

```yaml
services:
  worker:
    image: worker:latest
    x-quad-ops-init:
      - image: busybox:latest
        command: ["sh", "-c", "until nc -z redis 6379; do sleep 1; done"]
      - image: busybox:latest  
        command: ["sh", "-c", "until nc -z database 5432; do sleep 1; done"]
    depends_on:
      - redis
      - database
```

### Configuration Setup

```yaml
services:
  web:
    image: myapp:latest
    x-quad-ops-init:
      - image: busybox:latest
        command: ["sh", "-c", "echo 'server_name=web-$(hostname)' > /config/server.conf"]
      - image: envsubst:latest
        command: ["envsubst", "<", "/templates/app.conf.template", ">", "/config/app.conf"]
    environment:
      - SERVER_PORT=8080
      - DB_HOST=database
    volumes:
      - config:/config
      - ./templates:/templates:ro
```

## Advanced Examples

### Multiple Database Setup

```yaml
services:
  app:
    image: myapp:latest
    x-quad-ops-init:
      # Wait for databases to be ready
      - image: postgres:15
        command: ["pg_isready", "-h", "postgres", "-U", "user"]
      - image: redis:alpine
        command: ["redis-cli", "-h", "redis", "ping"]
      # Run migrations
      - image: migrate/migrate
        command: ["-path", "/migrations", "-database", "postgres://...", "up"]
      # Seed data
      - image: myapp:latest
        command: ["./scripts/seed-data.sh"]
    depends_on:
      - postgres
      - redis
```

### Build-Time and Runtime Initialization

```yaml
services:
  web:
    build: .
    x-quad-ops-init:
      # Download runtime dependencies
      - image: alpine:latest
        command: ["wget", "-O", "/shared/geoip.db", "https://example.com/geoip.db"]
      # Generate certificates
      - image: cfssl/cfssl
        command: ["cfssl", "gencert", "-config", "/certs/config.json", "/certs/csr.json"]
      # Validate configuration
      - image: myapp:latest
        command: ["./bin/validate-config", "/config/app.yaml"]
    volumes:
      - shared-data:/shared
      - certs:/certs
      - ./config:/config:ro
```

## Error Handling

### Failed Init Container

If an init container exits with a non-zero status:

```bash
# Check systemd status
systemctl --user status myproject-myservice-init-0.service

# View logs
journalctl --user -u myproject-myservice-init-0.service

# The main service will show as failed to start
systemctl --user status myproject-myservice.service
```

### Debugging

```bash
# Check all related units
systemctl --user list-units 'myproject-*'

# Reset failed services
systemctl --user reset-failed myproject-myservice-init-*.service

# Restart from the beginning
systemctl --user restart myproject-myservice.service
```

## Best Practices

### Keep Init Containers Lightweight

- Use minimal base images (busybox, alpine)
- Perform only essential initialization tasks
- Avoid long-running processes

### Idempotent Operations

Ensure init containers can run multiple times safely:

```yaml
x-quad-ops-init:
  - image: busybox:latest
    command: ["sh", "-c", "mkdir -p /data/cache || true"]
```

### Proper Error Handling

Exit with appropriate codes:

```yaml
x-quad-ops-init:
  - image: busybox:latest
    command: ["sh", "-c", "test -f /data/ready || { echo 'Not ready'; exit 1; }"]
```

### Resource Sharing

Use volumes for data exchange:

```yaml
services:
  app:
    image: myapp:latest
    x-quad-ops-init:
      - image: setup:latest
        command: "generate-config > /shared/app.conf"
    volumes:
      - config:/shared
volumes:
  config:
```

## Limitations

- Init containers run **sequentially**, not in parallel
- No direct communication between init containers
- Share data only through volumes or external services
- Each init container is a separate systemd service

## Related Documentation

- [Docker Compose Support](docker-compose-support) - Overview of compose features
- [Environment Files](environment-files) - Managing environment variables
- [Build Support](build-support) - Building custom images for init containers
