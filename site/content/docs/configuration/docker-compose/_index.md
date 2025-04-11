---
title: "Docker Compose"
weight: 0
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Docker Compose

Quad-Ops uses standard Docker Compose files (version 3.x format) for defining your container infrastructure. This allows you to use the familiar Docker Compose syntax while still benefiting from Podman's systemd integration via Quadlet.

## Example Docker Compose File

```yaml
version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - web-data:/usr/share/nginx/html
    networks:
      - frontend
    environment:
      - NGINX_HOST=example.com

volumes:
  web-data:

networks:
  frontend:
    driver: bridge
```

## Supported Resources

Quad-Ops can process the following resources from Docker Compose files:

### Services (Containers)

Services in Docker Compose are converted to Podman container units. The following properties are supported:

- `image`: Container image
- `ports`: Port mappings
- `volumes`: Volume mounts
- `networks`: Network connections
- `environment`: Environment variables
- `env_file`: Environment files
- `command`: Command to run
- `entrypoint`: Container entrypoint
- `user`: User to run as
- `working_dir`: Working directory
- `init`: Enable init process
- `privileged`: Run in privileged mode
- `read_only`: Mount root filesystem as read-only
- `security_opt`: Security options
- `hostname`: Set container hostname
- `secrets`: Secret mounts

## Conversion to Podman Container Units

When Quad-Ops processes a service definition from a Docker Compose file, it creates a corresponding Podman container unit with the following mapping:

| Docker Compose Property | Podman Container Property |
|-------------------------|---------------------------|
| `image` | `Image` |
| `ports` | `PublishPort` |
| `volumes` | `Volume` |
| `networks` | `Network` |
| `environment` | `Environment` |
| `env_file` | `EnvironmentFile` |
| `command` | `Exec` |
| `entrypoint` | `Entrypoint` |
| `user` | `User` |
| `working_dir` | `WorkingDir` |
| `init` | `RunInit` |
| `privileged` | `Privileged` |
| `read_only` | `ReadOnly` |
| `security_opt` | `SecurityLabel` |
| `hostname` | `HostName` |
| `labels` | `Label` |
| `secrets` | `Secrets` (array of Secret structs) |

Containers are created with systemd service files that ensure proper lifecycle management. The service name is used as the basis for the systemd unit name.

The `Secret` struct captures the following properties from Docker Compose secret definitions:
- `Source`: The name of the secret
- `Target`: The path where the secret is mounted in the container
- `UID`: User ID to set for the secret file
- `GID`: Group ID to set for the secret file
- `Mode`: File permissions for the secret file (defaults to "0644" if not specified)