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
| Option | Type | Description |
|-------------------|------|-------------|
| `name` | string | Unique identifier for the repository |
| `url` | string | Git repository URL to clone/pull from |
| `target` | string | Target commit or branch to checkout |
| `cleanup.action` | string | Cleanup policy (e.g., "keep", "delete") |

## Example Configuration

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
    target: main
    cleanup:
      action: keep
  - name: app2
    url: https://github.com/example/app2
    target: dev
    cleanup:
      action: delete
```
