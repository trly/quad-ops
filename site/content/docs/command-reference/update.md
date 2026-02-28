---
title: "update"
weight: 60
---

# quad-ops update

Update quad-ops to the latest version from GitHub releases.

## Synopsis

```
quad-ops update
```

## Description

The `update` command checks for the latest version of quad-ops from GitHub releases and downloads/applies the update if available.

## Example

```bash
quad-ops update
```

Output when update available:

```
Current version: v1.2.3
Checking for updates...
Update available! New version: v1.3.0
Downloading and applying update...
Update completed successfully! Please restart quad-ops to use the new version.
```

Output when already current:

```
Current version: v1.2.3
Checking for updates...
You are already running the latest version.
```

## Notes

- Requires internet connection to access GitHub releases
- May require `sudo` if installed system-wide
- Downloads from `https://github.com/trly/quad-ops/releases`
