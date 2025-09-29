---
title: "version"
weight: 80
---

# quad-ops version

Show version information for quad-ops.

## Synopsis

```
quad-ops version [flags]
```

## Options

```
  -h, --help   help for version
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

The `version` command displays detailed build information about the current quad-ops installation.

## Output Information

The command displays:

- Current version number
- Build commit SHA
- Build date and time
- Go version used for compilation
- Platform and architecture

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
