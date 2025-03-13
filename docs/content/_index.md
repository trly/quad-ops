+++
date = '2025-03-08T19:12:13-05:00'
draft = false
title = 'quad-ops Documentation'
+++

![quad-ops](/quad-ops/images/quad-ops.svg)

## GitOps for Quadlet Containers
![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg) ![Build](https://github.com/trly/quad-ops/actions/workflows/build-and-release.yml/badge.svg)

A lightweight GitOps framework for podman containers.

Quad-Ops is a tool that helps you manage container deployments using Podman and systemd in a GitOps workflow. It watches Git repositories for container definitions written in YAML and automatically converts them into systemd unit files that Podman can use to run your containers.

### Key Features:
- Monitors multiple Git repositories for container configurations
- Automatically generates systemd unit files from YAML definitions
- Supports containers, volumes, networks and images
- Handles automatic updates when Git repositories change
- Works in both system-wide and user modes

## Installation

### Installing from Source
To install quad-ops from source, you'll need to have Go installed on your system. Once you have Go installed, you can clone the quad-ops repository and build the binary:
```bash
git clone https://github.com/trly/quad-ops.git
cd quad-ops
go build
```
The binary will be built in the current directory. You can then move it to a directory in your PATH, such as /usr/local/bin, and make it executable:
```bash
sudo mv quad-ops /usr/local/bin/quad-ops
sudo chmod +x /usr/local/bin/quad-ops
```
### Installing a Pre-Built Binary
You can also download a pre-built binary for your platform from the [releases page](https://github.com/trly/quad-ops/releases). Simply download the appropriate binary for your system and make it executable.

## Configuration

### Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where repositories are stored |
| `syncInterval` | duration | `5m` | Interval between repository synchronization |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for quadlet files |
| `dbPath` | string | `/var/lib/quad-ops/quad-ops.db` | Path to the database file |
| `userMode` | boolean | `false` | Whether to run in user mode |
| `verbose` | boolean | `false` | Enable verbose logging |
| `repositories` | array | - | List of repositories to manage |

### Repository Options
| Option | Type | Description |
|-------------------|------|-------------|
| `name` | string | Unique identifier for the repository |
| `url` | string | Git repository URL to clone/pull from |
| `target` | string | Target directory within the repositoryDir |
| `cleanup.action` | string | Cleanup policy (e.g., "keep", "delete") |

### Example

```yaml
repositoryDir: /var/lib/quad-ops
syncInterval: 10m
quadletDir: /etc/containers/systemd
dbPath: /var/lib/quad-ops/quad-ops.db
userMode: false
verbose: true
repositories:
  - name: app1
    url: https://github.com/example/app1
    target: app1
    cleanup:
      action: keep
  - name: app2
    url: https://github.com/example/app2
    target: app2
    cleanup:
      action: delete
```

## Supported Unit Types

See `man systemd.unit(5)` or
[the offical Podman documentation](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
for more details on systemd unit options.

systemd unit documentation is available [here](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html)


### [Systemd](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `description` | string | - | Human-readable description |
| `documentation` | []string | - | Documentation URLs |
| `after` | []string | - | Units that must start before this one |
| `before` | []string | - | Units that must start after this one |
| `requires` | []string | - | Required dependencies |
| `wants` | []string | - | Optional dependencies |
| `conflicts` | []string | - | Units that cannot run alongside this one |
| `restart_policy` | string | - | Restart behavior (no, always, on-success, on-failure, on-abnormal, on-abort, on-watchdog) |
| `timeout_start_sec` | int | 0 | Timeout for starting the unit |
| `type` | string | - | Service type (simple, exec, forking, oneshot, notify) |
| `remain_after_exit` | bool | false | Keep service active after main process exits |
| `wanted_by` | []string | - | Target units that want this unit |

### Example

```yaml
systemd:
  # [Unit] section options
  description: "Human-readable description"
  documentation:
    - "https://example.com/docs"
    - "man:podman-container(5)"
  after:
    - "network.target"
  before:
    - "cleanup.service"
  requires:
    - "dependency.service"
  wants:
    - "optional-dependency.service"
  conflicts:
    - "incompatible.service"

  # [Service] section options
  type: "notify"  # simple, exec, forking, oneshot, notify
  restart_policy: "on-failure"  # no, always, on-success, on-failure, on-abnormal, on-abort, on-watchdog
  timeout_start_sec: 60
  remain_after_exit: true
  wanted_by:
    - "multi-user.target"
```

### [Container](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#container-units-container)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `image` | string | - | Container image to use |
| `label` | []string | - | Container labels (managed-by=quad-ops is added automatically) |
| `publish_port` | []string | - | Ports to publish (format: "host:container") |
| `environment` | map[string]string | - | Environment variables |
| `environment_file` | string | - | Path to file with environment variables |
| `volume` | []string | - | Volumes to mount |
| `network` | []string | - | Networks to connect to |
| `command` | []string | - | Command to run |
| `entrypoint` | []string | - | Container entrypoint |
| `user` | string | - | User to run as |
| `group` | string | - | Group to run as |
| `working_dir` | string | - | Working directory inside container |
| `podman_args` | []string | - | Additional arguments for podman |
| `run_init` | bool | false | Run an init inside the container |
| `notify` | bool | false | Container sends notifications to systemd |
| `privileged` | bool | false | Run container in privileged mode |
| `read_only` | bool | false | Mount root filesystem as read-only |
| `security_label` | []string | - | Security labels to apply |
| `host_name` | string | - | Hostname for the container |
| `secrets` | []SecretConfig | - | Secrets configuration |

### Container [Secret](https://docs.podman.io/en/latest/markdown/podman-secret-create.1.html)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | - | Secret name |
| `type` | string | - | Secret type |
| `target` | string | - | Target path |
| `uid` | int | 0 | User ID for ownership |
| `gid` | int | 0 | Group ID for ownership |
| `mode` | string | - | Permission mode |

### Example

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

### [Volume](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#volume-units-volume)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `label` | []string | - | Volume labels (managed-by=quad-ops is added automatically) |
| `device` | string | - | Device to mount |
| `options` | []string | - | Mount options |
| `uid` | int | 0 | User ID for ownership |
| `gid` | int | 0 | Group ID for ownership |
| `mode` | string | - | Permission mode |
| `chown` | bool | false | Change ownership to UID/GID |
| `selinux` | bool | false | Generate SELinux label |
| `copy` | bool | false | Copy contents from image |
| `group` | string | - | Volume group |
| `size` | string | - | Volume size |
| `capacity` | string | - | Volume capacity |
| `type` | string | - | Volume type |

### Example

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

### [Network](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#network-units-network)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `label` | []string | - | Network labels (managed-by=quad-ops is added automatically) |
| `driver` | string | - | Network driver |
| `gateway` | string | - | Gateway address |
| `ip_range` | string | - | IP address range |
| `subnet` | string | - | Subnet CIDR |
| `ipv6` | bool | false | Enable IPv6 |
| `internal` | bool | false | Restrict external access |
| `dns_enabled` | bool | false | Enable DNS |
| `options` | []string | - | Additional network options |

### Example
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

### [Image](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#image-units-image)

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `image` | string | - | Image to pull |
| `podman_args` | []string | - | Additional arguments for podman |

### Example

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

