---
title: "update"
weight: 60
---

# quad-ops update

Update quad-ops to the latest version from GitHub releases.

## Synopsis

```
quad-ops update [flags]
```

## Options

```
  -h, --help   help for update
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

The `update` command checks for and downloads the latest version of quad-ops from GitHub releases. It will automatically replace the current binary with the latest version if an update is available.

The update command:

1. **Version Check** - Compares current version against latest GitHub release
2. **Download** - Downloads latest binary if update is available
3. **Installation** - Replaces current binary with updated version
4. **Verification** - Confirms successful update completion

## Examples

### Check for and Install Updates

```bash
quad-ops update
```

### Update with verbose output

```bash
quad-ops update --verbose
```

### Sample Output

```bash
$ quad-ops update
Current version: v1.2.0
Checking for updates...
Update available! New version: v1.3.0
Downloading...
Update completed successfully!
```

### Already Up to Date

```bash
$ quad-ops update
Current version: v1.3.0
Checking for updates...
You are already running the latest version.
```

## Notes

### Requirements

- Internet connection to access GitHub releases
- Write permissions to the quad-ops binary location
- May require `sudo` if installed system-wide

### Update Process

The updater:

- Downloads binaries from `https://github.com/trly/quad-ops/releases`
- Verifies release authenticity through GitHub API
- Performs atomic replacement to avoid corruption
- Preserves existing file permissions

### Manual Update

If automatic update fails, manual installation is available:

```bash
# Download latest release manually
curl -L https://github.com/trly/quad-ops/releases/latest/download/quad-ops-linux-amd64 -o quad-ops
chmod +x quad-ops
sudo mv quad-ops /usr/local/bin/
```

