---
title: "image"
weight: 50
---

# quad-ops image

subcommands for managing and viewing images for quad-ops managed services

## Synopsis

```
quad-ops image [command]
```

## Available Commands

- **[pull](pull)** - pull an image from a registry

## Options

```
  -h, --help   help for image
```

## Global Options

```
      --config string           Path to the configuration file
  -o, --output string           Output format (text, json, yaml) (default "text")
      --quadlet-dir string      Path to the quadlet directory
      --repository-dir string   Path to the repository directory
  -u, --user                    Run in user mode
  -v, --verbose                 Enable verbose logging
```

## Description

The `image` command provides subcommands for managing container images used by quad-ops managed services. Currently supports pulling images from registries.

---

# quad-ops image pull

pull an image from a registry

## Synopsis

```
quad-ops image pull [flags]
```

## Options

```
  -h, --help   help for pull
```

## Global Options

```
      --config string           Path to the configuration file
  -o, --output string           Output format (text, json, yaml) (default "text")
      --quadlet-dir string      Path to the quadlet directory
      --repository-dir string   Path to the repository directory
  -u, --user                    Run in user mode
  -v, --verbose                 Enable verbose logging
```

## Description

The `image pull` command downloads container images referenced in your Docker Compose files. It discovers all images from configured repositories and pulls them automatically.

This command respects the `--user` flag for rootless operation and automatically sets the appropriate `XDG_RUNTIME_DIR` for user mode.

## Examples

### Pull All Images

Pull all images referenced in configured repositories:

```bash
sudo quad-ops image pull
```

### Rootless Mode

Pull images in user mode:

```bash
quad-ops --user image pull
```

### Verbose Output

Show detailed pull progress:

```bash
sudo quad-ops --verbose image pull
```

## Notes

- Images are pulled using Podman's native image management with default timeouts
- The command automatically discovers images from all configured repositories
- For rootless mode, ensure proper user namespace configuration
- Large images may take time to download; use `--verbose` to monitor progress
- For systemd unit timeout configuration during sync operations, see [Configuration](../../configuration/quad-ops-configuration)
