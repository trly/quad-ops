---
title: "Docker Compose"
weight: 10
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Docker Compose

Quad-Ops supports standard Docker Compose files (version 3.x) for defining your container infrastructure. When you add Docker Compose files to your Git repositories, Quad-Ops automatically converts them to Podman Quadlet unit files that systemd can use to run your containers.

## Supported File Names

Quad-Ops recognizes the following file names:
- `docker-compose.yml`
- `docker-compose.yaml`
- `compose.yml`
- `compose.yaml`

## Project Naming Convention

Quad-Ops generates project names automatically based on the repository structure:
- Format: `<repo>-<folder>`
- Example: `test-photoprism` for a compose file in repositories/home/test/photoprism

This naming convention ensures proper DNS resolution between containers in the same project.

## Example Docker Compose File

```yaml
version: '3.8'

services:
  web:
    image: docker.io/nginx:latest
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

## Supported Features

Quad-Ops can process the following resources from Docker Compose files:

### Container Features

Services in Docker Compose are converted to Podman container units. The following properties are supported:

- `image`: Container image (use fully qualified names with registry prefix)
- `ports`: Port mappings
- `volumes`: Volume mounts
- `networks`: Network connections
- `environment`: Environment variables
- `env_file`: Environment files
- `command`: Command to run
- `entrypoint`: Container entrypoint
- `user`: User to run as
- `working_dir`: Working directory
- `init`: Enable init process (`init: true/false`)
- `read_only`: Read-only container filesystems
- `depends_on`: Container startup order with systemd dependency conversion
- `hostname`: Custom hostnames
- `secrets`: File-based secrets
- `labels`: Container labels

### Network Features
- **Custom Networks**: Create isolated networks
- **Network Drivers**: Bridge, host, and other supported drivers
- **IPAM Settings**: Subnet, gateway, IP ranges
- **IPv6 Support**: Enable IPv6 networking
- **Internal Networks**: Create internal-only networks
- **Network Labels**: Add metadata to networks
- **Driver Options**: Pass options to network driver

### Volume Features
- **Named Volumes**: Persistent storage with names
- **Bind Mounts**: Mount host directories
- **Volume Options**: Driver-specific options

## Conversion to Podman Container Units

When Quad-Ops processes a service definition from a Docker Compose file, it creates a corresponding Podman container unit with the following mapping. Note that some Docker Compose properties like `privileged` and `security_opt` are not supported by Podman Quadlet:

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
| `read_only` | `ReadOnly` |
| `depends_on` | Used for systemd unit dependencies |
| `hostname` | `HostName` |
| `labels` | `Label` |
| `secrets` | `Secrets` (array of Secret structs) |

Containers are created with systemd service files that ensure proper lifecycle management. The service name is used as the basis for the systemd unit name. Container dependencies defined with `depends_on` in Docker Compose are converted to systemd unit dependencies using `After` and `Requires` directives.

The `Secret` struct captures the following properties from Docker Compose secret definitions:
- `Source`: The name of the secret
- `Target`: The path where the secret is mounted in the container
- `UID`: User ID to set for the secret file
- `GID`: Group ID to set for the secret file
- `Mode`: File permissions for the secret file (defaults to "0644" if not specified)

## Unsupported Features

The following Docker Compose features are not supported by Podman Quadlet:

- **Privileged Mode**: Not directly supported (use specific capabilities instead)
- **SecurityLabel**: Use alternative Podman security options
- **DNSEnabled** in networks: Configure via driver options instead
- **Swarm Mode**: Podman doesn't support Swarm-specific features
- **Docker-specific Extensions**: Custom Docker-specific fields
- **Health Checks**: Not directly supported in Quadlet (configured differently)
- **Service Discovery**: Limited to systemd-based DNS resolution
- **Deploy Config**: Swarm-specific deployment settings

## Podman Quadlet Best Practices for Docker Compose Files

- Always use fully qualified image names with registry prefix (docker.io/, quay.io/, etc.)
- Container dependencies must use the service name format in systemd unit files
- Use After/Requires with .service suffix (e.g., 'After=db.service', not 'After=db.container')
- By default, quad-ops provides predictable container hostnames without the `systemd-` prefix that Podman normally adds (see [Docker Compose Networking](/docs/configuration/docker-compose/docker-compose-networking/) for details)
- Quadlet does not auto-create bind mount directories - they must exist before container start
