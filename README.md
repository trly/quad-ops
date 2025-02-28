# Quad-Ops

Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from YAML manifests and handles unit reloading.

## Features

- Multi-repository support with independent branch/tag/commit targeting
- Automatic systemd unit generation and reloading
- Daemon mode with configurable check intervals
- User mode support for non-root operation
- Content-aware updates (only updates changed units)
- Verbose logging option for debugging
- Dry-run capability for testing configurations

## Installation

1. Build the binary:

```bash
go build -o quad-ops main.go
```

2. Create required directories:
```bash
mkdir -p /etc/quad-ops
```

3. Copy the binary, configuration file, and systemd service file:
```bash
cp quad-ops /usr/local/bin/
cp configs/config.yaml /etc/quad-ops/
cp build/package/quad-ops.service /etc/systemd/system/
```

4. Enable and start the service:
```bash
systemctl enable quad-ops
systemctl start quad-ops
```

## Usage
Run as a one-time check:
```bash
quad-ops --config /etc/quad-ops/config.yaml
```

Run as a daemon with a 5-minute check interval:
```bash
quad-ops --daemon --interval 300
```

Run in systemd user-mode:
```bash
quad-ops --user-mode
```

## Configuration

The default configuration file location is `/etc/quad-ops/config`
```yaml
---
repositories:
  # Uses root of repository
  - name: "app1"
    url: "https://github.com/org/app1.git"
    target: "main"

  # Uses specific manifest directory
  - name: "platform-monorepo"
    url: "https://github.com/org/platform.git"
    target: "main"
    manifest_dir: "hosts/prod-cluster-1/manifests"

paths:
  repository_dir: "/var/lib/quad-ops/repos"
  quadlet_dir: "/etc/containers/systemd"
```

## Supported Unit Types

### Containers

```yaml
---
name: web-app
type: container
systemd:
  description: "Web application"
  after: ["network.target"]
  restart_policy: "always"
  timeout_start_sec: 30
container:
  image: "nginx:latest"
  publish: ["8080:80"]
  environment:
    NGINX_PORT: "80"
  environment_file: "/etc/nginx/env"
  volume: ["/data:/app/data"]
  network: ["app-network"]
  secrets:
    - name: "web-cert"
      type: "mount"
      target: "/certs"
      mode: "0400"
```

### Volumes

```yaml
---
name: data-vol
type: volume
systemd:
  description: "Data volume"
volume:
  label: ["environment=prod"]
  device: "/dev/sda1"
  options: ["size=10G"]
  uid: 1000
  gid: 1000
  mode: "0755"
```

### Networks

```yaml
---
name: app-net
type: network
systemd:
  description: "Application network"
network:
  driver: "bridge"
  subnet: "172.20.0.0/16"
  gateway: "172.20.0.1"
  ipv6: true
  dns_enabled: true
```

### Images

```yaml
---
name: app-image
type: image
systemd:
  description: "Application image"
image:
  image: "registry.example.com/app:latest"
  podman_args: ["--tls-verify=false"]
```
