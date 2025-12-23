---
title: "version"
weight: 80
---

# quad-ops version

Show version information for quad-ops.

## Synopsis

```
quad-ops version
```

## Description

The `version` command displays build information about the current quad-ops installation and checks for available updates.

## Output

- Version number
- Build commit SHA
- Build date
- Go version used for compilation

## Example

```bash
quad-ops version
```

Output:

```
quad-ops version v1.2.3
  commit: abc123def456
  built: 2024-01-15T10:30:00Z
  go: go1.23.0

Checking for updates...
You are using the latest version.
```
