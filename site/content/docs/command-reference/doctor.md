---
title: "doctor"
weight: 40
---

# quad-ops doctor

Check system health and configuration for quad-ops.

The doctor command performs comprehensive checks of:

- System requirements (systemd, podman)
- Configuration file validity
- Directory permissions and accessibility
- Repository connectivity
- File system requirements

This helps diagnose common setup and configuration issues.

## Synopsis

```
quad-ops doctor [flags]
```

## Options

```
  -h, --help   help for doctor
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

## Check Categories

### System Requirements

- **systemd availability** - Verifies systemd is available and functional
- **Podman installation** - Checks for proper Podman installation
- **User mode compatibility** - Validates rootless container support (when using `--user`)

### Configuration Validation

- **Config file syntax** - Validates YAML/TOML configuration format
- **Repository definitions** - Checks repository URLs and credentials
- **Directory paths** - Verifies configured directories exist and are accessible

### Permissions and Access

- **Directory permissions** - Ensures quad-ops can read/write to required directories
- **Git repository access** - Tests connectivity to configured Git repositories
- **systemd user access** - Validates systemd user service permissions (user mode)

### File System Requirements

- **Disk space** - Checks available disk space for quadlet units and repositories
- **Path accessibility** - Verifies all configured paths are accessible
- **Lock file creation** - Tests ability to create synchronization lock files

## Examples

### Basic health check

```bash
quad-ops doctor
```

### Check with verbose output

```bash
quad-ops doctor --verbose
```

### Check user mode configuration

```bash
quad-ops doctor --user
```

### Check with custom configuration

```bash
quad-ops doctor --config /path/to/config.yaml
```

## Output Formats

The doctor command supports different output formats for integration with monitoring systems:

### Text Output (default)

```bash
quad-ops doctor
# ✓ systemd is available
# ✓ Podman is installed (version 4.9.0)
# ✓ Configuration file is valid
# ✗ Repository 'my-app' is not accessible
```

### JSON Output

```bash
quad-ops doctor --output json
# {"checks": [{"name": "systemd", "status": "pass"}, ...]}
```

### YAML Output

```bash
quad-ops doctor --output yaml
# checks:
#   - name: systemd
#     status: pass
```

## Troubleshooting

Common issues identified by the doctor command:

### systemd Not Available

- Install systemd or use a systemd-compatible system
- For user mode: enable lingering with `loginctl enable-linger`

### Podman Not Found

- Install podman package for your distribution
- Ensure podman is in your PATH

### Permission Denied

- Check directory ownership and permissions
- For user mode: ensure directories are owned by the current user
- For system mode: ensure directories are accessible by root

### Repository Access Issues

- Verify Git repository URLs are correct
- Check network connectivity to repository hosts
- Ensure proper authentication (SSH keys, tokens) is configured
