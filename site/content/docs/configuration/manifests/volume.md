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
| `containers_conf_module` | []string | - | Module settings |
| `driver` | string | - | Volume driver |
| `global_args` | []string | - | Global arguments |
| `image` | string | - | Image to use |
| `podman_args` | []string | - | Additional podman arguments |
| `volume_name` | string | - | Volume name |
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
  driver: "local"
  copy: true
  group: "storage"
  type: "block"
```