---
title: "Configuration"
weight: 20
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---
# Configuration

## Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `repositoryDir` | string | `/var/lib/quad-ops` | Directory where repositories are stored |
| `syncInterval` | duration | `5m` | Interval between repository synchronization |
| `quadletDir` | string | `/etc/containers/systemd` | Directory for quadlet files |
| `dbPath` | string | `/var/lib/quad-ops/quad-ops.db` | Path to the database file |
| `userMode` | boolean | `false` | Whether to run in user mode |
| `verbose` | boolean | `false` | Enable verbose logging |
| `repositories` | array | - | List of repositories to manage |

## Repository Options
| Option | Type | Default | Description |
|-------------------|------|---------|-------------|
| `name` | string | - | Unique identifier for the repository |
| `url` | string | - | Git repository URL to clone/pull from |
| `ref` | string | - | Git reference to checkout (branch, tag, or commit hash) |
| `manifestDir` | string | "" | Subdirectory within repo where manifests are located |
| `cleanup` | string | "keep" | Cleanup policy: "keep" or "delete" |

## Example Configuration

```yaml
# Global settings
repositoryDir: /var/lib/quad-ops
syncInterval: 10m
quadletDir: /etc/containers/systemd
dbPath: /var/lib/quad-ops/quad-ops.db
userMode: false
verbose: true

# Repository definitions
repositories:
  - name: app1
    url: https://github.com/example/app1
    ref: main
    manifestDir: manifests
    cleanup: keep  # Units remain even if removed from manifests
    
  - name: app2
    url: https://github.com/example/app2
    ref: dev
    cleanup: delete  # Units are stopped and removed when they're no longer in manifests
```
