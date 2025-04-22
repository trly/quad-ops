---
title: "Getting Started"
weight: 5
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

# Getting Started with Quad-Ops

This guide walks you through setting up Quad-Ops and syncing your first Docker Compose file, providing a smooth transition from Docker Compose commands to Quad-Ops.

## Prerequisites

- Podman 4.0+ installed
- Git installed
- systemd-based Linux distribution

## Quick Start (5 Minutes)

### Step 1: Install Quad-Ops

Download and install the latest release from GitHub: https://github.com/trly/quad-ops/releases

```bash
wget https://github.com/trly/quad-ops/releases/download/v0.3.0/quad-ops_0.3.0_linux_amd64.tar.gz
tar -xzf quad-ops_0.3.0_linux_amd64.tar.gz
sudo mv quad-ops /usr/local/bin/
sudo chmod +x /usr/local/bin/quad-ops
```

### Alternative: Install from Source

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git

# Change to the repo directory
cd quad-ops

# Build the binary
go build -o quad-ops cmd/quad-ops/main.go

# Move to system directory
sudo mv quad-ops /usr/local/bin/
```

### Step 2: Create a Basic Configuration

```bash
# Create configuration directory
sudo mkdir -p /etc/quad-ops
```

Create a basic config file at `/etc/quad-ops/config.yaml`:

```yaml
repositories:
  - name: quad-ops
    url: "https://github.com/trly/quad-ops.git"
    ref: "main"
    composeDir: "examples"
```

### Step 3: Run Your First Sync

```bash
quad-ops sync
```

## Verifying Your Setup

### Check quad-ops status

```bash
quad-ops unit list -t all

ID  Name                                Type       Unit State  SHA1                                      Cleanup Policy  Created At
1   quad-ops-multi-service-db           container  active      c79f25a54e5aca33d8bdf7e4b4776969959aa4b4  keep            2025-04-21 22:45:15 +0000 UTC
2   quad-ops-multi-service-webapp       container  active      106a63b255e897348957b4b2cee17a6e9e4d0e00  keep            2025-04-21 22:45:15 +0000 UTC
3   quad-ops-multi-service-db-data      volume     active      05763d60c00d6ef3f4f8a026083877eb6545c48b  keep            2025-04-21 22:45:15 +0000 UTC
4   quad-ops-multi-service-wp-content   volume     active      05763d60c00d6ef3f4f8a026083877eb6545c48b  keep            2025-04-21 22:45:15 +0000 UTC
5   quad-ops-multi-service-app-network  network    active      479a643178b4bb4d2fdd8d6193c749e34c35ce83  keep            2025-04-21 22:45:15 +0000
```

### Check container status

```bash
podman ps

CONTAINER ID  IMAGE                               COMMAND               CREATED      STATUS      PORTS                 NAMES
a31ba0448047  docker.io/library/mariadb:latest    mariadbd              3 hours ago  Up 3 hours  3306/tcp              quad-ops-multi-service-db
731cd5df42ff  docker.io/library/wordpress:latest  apache2-foregroun...  3 hours ago  Up 3 hours  0.0.0.0:8080->80/tcp  quad-ops-multi-service-webapp
```

## Understanding the Workflow

1. **Create Docker Compose Files**: Create standard Docker Compose files in your Git repository
2. **Configure Quad-Ops**: Tell Quad-Ops where to find your repositories
3. **Sync Repositories**: Quad-Ops pulls your repos and converts Compose files to systemd units
4. **Manage with systemd**: Use standard systemd commands to start/stop/monitor services

## Running as a Service

For production use, set up Quad-Ops as a systemd service for continuous operation. See the [Systemd Service Configuration](/quad-ops/docs/configuration/systemd-service/) guide for detailed instructions.

## Key Differences from Docker Compose

When transitioning from Docker Compose to Quad-Ops + Podman, be aware of these important differences:

1. **Container Names & DNS Resolution**:
   - Default format (with `usePodmanDefaultNames: false`): `projectname-servicename`
   - Alternative format (with `usePodmanDefaultNames: true`): `systemd-projectname-servicename`
   - **NetworkAlias**: Services can reference each other using just the service name (e.g., `db` instead of full hostname)

2. **Image Names**:
   - Always use fully qualified image names with registry prefix
   - Bad: `image: nginx`
   - Good: `image: docker.io/library/nginx:latest`

3. **Bind Mounts**:
   - Directories must exist on the host before container start (not auto-created)
      - Management of these is outside the scope of Quad-Ops and should be handled by a
      separate orchestration tool if automation is desired.
   - Absolute paths are recommended for clarity

4. **Unsupported Features**:
   - Privileged mode: Not supported by Podman Quadlet
   - SecurityLabel: Not supported, use specific security options instead
   - DNSEnabled in networks: Not directly supported
   - Docker Swarm features: Not supported by Podman
   - See [Docker Compose Support](/quad-ops/docs/configuration/docker-compose/) for details



## Troubleshooting Common Issues

### Container Won't Start

```bash
# Check the service status
systemctl status my-apps-webapp.service

# View detailed logs
journalctl -u my-apps-webapp.service
```

Common issues include:
- Missing bind mount directories
- Incomplete image names
- Permission issues with volume mounts

### Container Networking Problems

If containers can't communicate:
- Check network unit status: `systemctl status my-apps-default.network`
- Verify container DNS resolution format matches your configuration (default: `my-apps-servicename`, with usePodmanDefaultNames: `systemd-my-apps-servicename`)
- Check network configuration in systemd unit file: `systemctl cat my-apps-default.network`

## Next Steps

- [Configure for Production](/quad-ops/docs/configuration/systemd-service/)
- [Explore Docker Compose Support](/quad-ops/docs/configuration/docker-compose/)
- [Implement Secrets Management](/quad-ops/docs/configuration/docker-compose/secrets/)
- [See Example Configurations](/quad-ops/docs/configuration/examples/)
