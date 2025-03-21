---
title: "Manifests"
weight: 0
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Manifests

quad-ops manifests are a simplified representation of [systemd quadlet units](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html).


```yaml
---
name: quad-ops-example
type: container
systemd:
  description: quad-ops example manifest
  documentation:
    - https://github.com/trly/quad-ops
  after:
    - network-online.target
  before:
    - shutdown.target
  restart_policy: always
  timeout_start_sec: 300
  type: notify
  wanted_by:
    - default.target
  volume: quad-ops-example.volume:/data
container:
  image: quad-ops-demo.image
  label:
    - "environment=staging"
  publish:
    - "8080:80"
  network:
    - "quad-ops-example.network"
  network_mode: bridge
---
name: quad-ops-demo
type: volume
volume:
  label:
    - "usage=quad-ops-data"
    - "environment=staging"Z
---
name: quad-ops-demo
type: network
network:
  label:
    - "environment=staging"
  driver: bridge
---
name: quad-ops-demo
type: image
image:
  label:
    - "environment=staging"
  image: docker.io/traefik/whoami:latest
```

## Systemd

https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html

> The following options are available for all supported quadlet unit types

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

## Example

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
