---
title: "Getting Started"
weight: 2
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

After syncing, check the status of your units:

```bash
# Check system journal for sync logs
journalctl -u quad-ops.service -f

# View unit file contents
systemctl cat my-apps-webapp.service

# Check container status
podman ps
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
   - Format: `systemd-projectname-servicename` (not `servicename` like in Docker Compose)
   - Example: In your code, reference `systemd-myapp-db` not just `db`

2. **Image Names**:
   - Always use fully qualified image names with registry prefix
   - Bad: `image: nginx`
   - Good: `image: docker.io/nginx:latest`

3. **Bind Mounts**:
   - Directories must exist on the host before container start (not auto-created)
   - Absolute paths are recommended for clarity

4. **Unsupported Features**:
   - Privileged mode: Use specific capabilities instead
   - Docker Swarm features: Not supported by Podman
   - See [Docker Compose Support](/docs/docker-compose/) for details



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
- Verify container DNS resolution format: `systemd-my-apps-servicename`
- Check network configuration in systemd unit file: `systemctl cat my-apps-default.network`

## Next Steps

- [Configure for Production](/docs/configuration/systemd-service/)
- [Explore Docker Compose Support](/docs/docker-compose/)
- [Implement Secrets Management](/docs/configuration/docker-compose/secrets/)
- [See Example Configurations](/docs/examples/)