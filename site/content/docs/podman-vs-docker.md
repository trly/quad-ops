---
title: "Podman vs Docker Compose"
weight: 5
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

# Podman vs Docker Compose: Key Differences

While Quad-Ops allows you to use Docker Compose files with Podman, there are important differences you should understand. This guide explains the key differences and how to adapt your Docker Compose files for Podman Quadlet.

## Conceptual Differences

| Docker Compose | Podman Quadlet |
|----------------|----------------|
| Standalone orchestration tool | Integration with systemd |
| Docker-specific features | Daemonless, rootless architecture |
| YAML-driven configuration | Unit files with systemd integration |
| Single `docker-compose.yml` | Multiple `.container`, `.volume`, `.network` files |
| Built-in dependency resolution | systemd-based dependency management |

## Container Naming and DNS Resolution

### Docker Compose
```yaml
# Docker Compose example
services:
  db:
    image: postgres
  webapp:
    image: nginx
    depends_on:
      - db
    # In code, you would connect to "db"
    environment:
      - DB_HOST=db
```

### Podman Quadlet
```yaml
# Podman-compatible Compose file
services:
  db:
    image: docker.io/postgres:latest
  webapp:
    image: docker.io/nginx:latest
    depends_on:
      - db
    # Must use systemd-prefixed names
    environment:
      - DB_HOST=systemd-myproject-db
```

## Image References

### Docker Compose (Implicit Registry)
```yaml
services:
  app:
    image: nginx:latest   # Works in Docker
```

### Podman Quadlet (Explicit Registry Required)
```yaml
services:
  app:
    image: docker.io/nginx:latest  # Required in Podman
```

## Bind Mounts

### Docker Compose (Auto-creates Directories)
```yaml
services:
  app:
    volumes:
      - ./data:/app/data  # Directory created if missing
```

### Podman Quadlet (Must Pre-create Directories)
```bash
# You must manually create this directory first
mkdir -p ./data
```
```yaml
services:
  app:
    volumes:
      - ./data:/app/data  # Directory must exist
```

## Environment Variables

### Docker Compose
```yaml
services:
  app:
    environment:
      - DEBUG=true
      - SECRET_KEY=${SECRET_KEY}  # From .env or shell
```

### Podman Quadlet
```yaml
services:
  app:
    environment:
      - DEBUG=true
    env_file:
      - .env  # Better to use env_file for secrets
```

## Security Features

### Docker Compose (Privileged Mode)
```yaml
services:
  app:
    privileged: true  # Works in Docker
```

### Podman Quadlet (Specific Capabilities)
```yaml
services:
  app:
    # Use specific capabilities instead
    cap_add:
      - SYS_ADMIN
      - NET_ADMIN
```

## Service Dependencies

### Docker Compose
```yaml
services:
  webapp:
    depends_on:
      - db
      - redis
```

### Podman Quadlet (Converted to systemd Dependencies)
```ini
# Generated unit file (myproject-webapp.container)
[Unit]
Requires=myproject-db.service myproject-redis.service
After=myproject-db.service myproject-redis.service
```

## Restart Policies

### Docker Compose
```yaml
services:
  app:
    restart: always
```

### Podman Quadlet (systemd Restart)
```ini
# Generated unit file
[Service]
Restart=always
```

## Secrets

### Docker Compose (Swarm Secrets)
```yaml
services:
  app:
    secrets:
      - db_password
secrets:
  db_password:
    external: true  # Managed by Swarm
```

### Podman Quadlet (File-based Secrets)
```yaml
services:
  app:
    secrets:
      - source: db_password
        target: /run/secrets/db_password
secrets:
  db_password:
    file: ./secrets/db_password.txt  # Must be a file path
```

## Commands and Management

| Docker Compose | Podman Quadlet |
|----------------|----------------|
| `docker-compose up -d` | `systemctl start project-service.service` |
| `docker-compose down` | `systemctl stop project-service.service` |
| `docker-compose logs` | `journalctl -u project-service.service` |
| `docker-compose ps` | `podman ps` |
| `docker-compose build` | `podman build` + update unit files |

## Compatibility Tips

1. **Use Fully Qualified Image Names**: Always include the registry prefix (e.g., `docker.io/`)

2. **Pre-create Directories**: Create all bind mount directories before starting containers

3. **Use systemd-prefixed DNS Names**: Replace direct service references with `systemd-project-service`

4. **Remove Unsupported Features**: Avoid privileged mode, DNSEnabled, SecurityLabel

5. **Use File-based Secrets**: Configure secrets as files with proper permissions

6. **Include Default Network**: Always include network configuration, even for default networks