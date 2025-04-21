---
title: "Docker Compose Support"
weight: 0
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

# Docker Compose Support

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

## Supported Features

### Container Features
- **Images**: Fully qualified image names (docker.io/, quay.io/, etc.)
- **Ports**: Host to container port mapping
- **Environment Variables**: Including env_file support
- **Volumes**: Both named volumes and bind mounts
- **Networks**: Custom networks with various configuration options
- **Command/Entrypoint**: Override default container command
- **User/WorkingDir**: Set container user and working directory
- **Init Process**: Enable/disable init process (`init: true/false`)
- **Read-only**: Read-only container filesystems
- **Hostname**: Custom hostnames
- **Dependencies**: Container startup order via `depends_on`
- **Secrets**: File-based secrets (see [Podman Secrets](/docs/podman-secrets) for details)

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

## Podman Quadlet Best Practices

- Always use fully qualified image names with registry prefix (docker.io/, quay.io/, etc.)
- Container dependencies must use the service name format in systemd unit files
- Use After/Requires with .service suffix (e.g., 'After=db.service', not 'After=db.container')
- By default, quad-ops provides predictable container hostnames without the `systemd-` prefix that Podman normally adds (see [Docker Compose Networking](/docs/docker-compose-networking/) for details)
- Named volumes require the '.volume' suffix in Volume= directives
- Quadlet does not auto-create bind mount directories - they must exist before container start