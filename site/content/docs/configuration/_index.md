---
title: "Configuration"
weight: 10
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---
# Configuration

Quad-Ops uses Docker Compose files for defining your container infrastructure. Standard docker-compose.yml files that define services, networks, volumes, and secrets are automatically processed and converted to Podman Quadlet units.

## Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where repositories are stored |
| `syncInterval` | duration | `5m` | Interval between repository synchronization |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for quadlet files |
| `dbPath` | string | `/var/lib/quad-ops/quad-ops.db` | Path to the database file |
| `userMode` | boolean | `false` | Whether to run in user mode |
| `verbose` | boolean | `false` | Enable verbose logging |
| `usePodmanDefaultNames` | boolean | `false` | Whether to use Podman's default container naming with systemd- prefix |
| `repositories` | array | - | List of repositories to manage |

## Repository Options
| Option | Type | Default | Description |
|-------------------|------|---------|-------------|
| `name` | string | - | Unique identifier for the repository |
| `url` | string | - | Git repository URL to clone/pull from |
| `ref` | string | - | Git reference to checkout (branch, tag, or commit hash) |
| `composeDir` | string | "" | Subdirectory within repo where Docker Compose files are located |
| `cleanup` | string | "keep" | Cleanup policy: "keep" or "delete" |
| `usePodmanDefaultNames` | boolean | `false` | Whether to use Podman's default naming for this repository (overrides global setting) |

### Cleanup Policy Options

- `keep` (default): Units from this repository remain deployed even when the compose file is removed
- `delete`: Units that no longer exist in the repository Docker Compose files will be stopped and removed

## Example Configuration

```yaml
# Global settings
repositoryDir: /var/lib/quad-ops
syncInterval: 10m
quadletDir: /etc/containers/systemd
dbPath: /var/lib/quad-ops/quad-ops.db
userMode: false
verbose: true
usePodmanDefaultNames: false  # No systemd- prefix in container hostnames

# Repository definitions
repositories:
  - name: app1
    url: https://github.com/example/app1
    ref: main
    composeDir: compose
    cleanup: keep  # Units remain even if removed from Docker Compose files

  - name: app2
    url: https://github.com/example/app2
    ref: dev
    cleanup: delete  # Units are stopped and removed when they're no longer in Docker Compose files
    usePodmanDefaultNames: true  # Use systemd- prefix in container hostnames for this repo only
```
