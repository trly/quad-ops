---
title: "update"
weight: 30
---

# update

Update quad-ops to the latest version from GitHub releases.

## Synopsis

The `update` command checks for and downloads the latest version of quad-ops from GitHub releases. It will automatically replace the current binary with the latest version if an update is available.

```
quad-ops update [flags]
```

## Description

The update command:

1. **Version Check** - Compares current version against latest GitHub release
2. **Download** - Downloads latest binary if update is available
3. **Installation** - Replaces current binary with updated version
4. **Verification** - Confirms successful update completion

## Examples

### Check for and Install Updates
```bash
# Update to latest version
quad-ops update
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

## Options

The update command uses only global options:

| Option | Short | Description |
|--------|-------|-------------|
| `--help` | `-h` | Show help for update command |
| `--verbose` | `-v` | Enable verbose output |

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

### Security Considerations
- Updates are fetched from official GitHub releases only
- Binary signatures are verified where available
- Automatic updates can be disabled by restricting network access

## Related Commands

- [`version`](../version) - Show current version information
- [`config`](../config) - Manage configuration for automatic updates

### Manual Update
If automatic update fails, manual installation is available:
```bash
# Download latest release manually
curl -L https://github.com/trly/quad-ops/releases/latest/download/quad-ops-linux-amd64 -o quad-ops
chmod +x quad-ops
sudo mv quad-ops /usr/local/bin/
```