---
title: "image pull"
weight: 45
---

# image pull

Pull container images from configured repositories.

## Synopsis

```
quad-ops image pull
```

## Description

The `image pull` command downloads container images referenced in your Docker Compose files. It discovers all images from configured repositories and pulls them automatically.

This command respects the `--user` flag for rootless operation and automatically sets the appropriate `XDG_RUNTIME_DIR` for user mode.

## Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output showing pull progress |
| `--user` | `-u` | Pull images in rootless user mode |
| `--quadlet-dir` | | Override unit output directory |
| `--repository-dir` | | Override git checkout directory |

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

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | Success - all images pulled successfully |
| `1` | General error during image pull |
| `2` | Invalid command usage |
| `3` | Configuration error |

## Related Commands

- [sync](../sync) - Synchronize repositories (includes image pulling)
- [Configuration](../../configuration/quad-ops-configuration) - Configure image pull settings
