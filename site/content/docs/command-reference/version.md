---
title: "version"
weight: 60
---

# version

Display version information and check for updates.

## Synopsis

```
quad-ops version
```

## Description

The `version` command displays detailed build information about the current quad-ops installation and checks for available updates from GitHub releases.

## Output Information

The command displays:
- Current version number
- Build commit SHA
- Build date and time
- Go version used for compilation
- Platform and architecture
- Available updates (if any)

## Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |
| `--user` | `-u` | Run in rootless user mode |
| `--quadlet-dir` | | Override unit output directory |
| `--repository-dir` | | Override git checkout directory |

## Examples

### Display Version Information
```bash
quad-ops version
```

Example output:
```
quad-ops version v1.2.3
Built: 2024-01-15T10:30:00Z
Commit: abc123def456
Go version: go1.21.0
OS/Arch: linux/amd64

Checking for updates...
Latest version: v1.2.4 (update available)
```

### Check Version in Scripts
The version command returns appropriate exit codes for scripting:
```bash
if quad-ops version | grep -q "update available"; then
    echo "Update available"
fi
```

## Update Check

The version command automatically checks GitHub releases for newer versions. This requires internet connectivity but will gracefully handle offline scenarios.

## Privacy

The version check makes a simple HTTP request to GitHub's public API and does not transmit any personal or system information beyond what's included in standard HTTP headers.

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | Success - version information displayed |
| `1` | Error retrieving version information |

## Related Commands

- [update](../update) - Update quad-ops to the latest version
- [Releases](../../releases) - View release history and changelog
