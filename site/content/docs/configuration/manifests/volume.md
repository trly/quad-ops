---
title: "Volume"
weight: 40
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Volume

## Options

> https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#volume-units-volume

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

## Example

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