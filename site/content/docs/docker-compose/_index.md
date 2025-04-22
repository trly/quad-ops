---
title: "Docker Compose"
weight: 10
bookCollapseSection: true
---

# Docker Compose Support

Quad-Ops uses Docker Compose files as the configuration format for defining your container infrastructure. When you add Docker Compose files to your Git repositories, Quad-Ops automatically converts them to Podman Quadlet unit files that systemd manages.

## File Detection and Naming

Quad-Ops automatically detects these file names:
- `docker-compose.yml`
- `docker-compose.yaml`
- `compose.yml`
- `compose.yaml`

Project names are generated based on repository structure:
- Format: `<repo>-<folder>`
- Example: `test-photoprism` for repositories/home/test/photoprism

## Key Differences from Docker Compose

| Difference | Description |
|------------|-------------|
| **Container Naming** | By default, container hostnames match their service names without the systemd- prefix |
| **Image References** | Always use fully qualified image names (docker.io/library/nginx:latest) |
| **Bind Mounts** | Directories must exist before containers start (not auto-created) |
| **Service Discovery** | Simple service names (db) work as DNS names regardless of actual container hostname |
| **Management** | Containers are managed via systemd commands, not docker-compose commands |

## Component Documentation

Each Docker Compose component is converted to a corresponding Podman resource:

- [Services](services) - Container configuration
- [Networks](networks) - Container networking
- [Volumes](volumes) - Persistent storage
- [Secrets](secrets) - Sensitive data management
- [Dependency Management](../dependency-management) - How service relationships are handled