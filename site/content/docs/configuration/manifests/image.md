---
title: "Image"
weight: 20
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# Image

## Options

> https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#image-units-image

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `image` | string | - | Image to pull |
| `podman_args` | []string | - | Additional arguments for podman |

## Example

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
