---
title: "Container"
weight: 10
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Container

## Options

> https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#container-units-container

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `image` | string | - | Container image to use |
| `label` | []string | - | Container labels (managed-by=quad-ops is added automatically) |
| `publish` | []string | - | Ports to publish (format: "host:container") |
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

## Container Secret

> https://docs.podman.io/en/latest/markdown/podman-secret-create.1.html

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | - | Secret name |
| `type` | string | - | Secret type |
| `target` | string | - | Target path |
| `uid` | int | 0 | User ID for ownership |
| `gid` | int | 0 | Group ID for ownership |
| `mode` | string | - | Permission mode |

## Example

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