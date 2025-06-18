---
title: "config"
weight: 50
---

# config

Display current configuration.

## Synopsis

```
quad-ops config [OPTIONS]
```

## Description

The `config` command displays the current configuration for quad-ops, including all defaults and any overrides. The output is formatted as YAML for easy inspection and debugging.

This command is useful for:
- Verifying configuration settings
- Debugging configuration issues  
- Understanding active configuration values
- Checking default values and overrides

## Examples

```bash
# Display current configuration
quad-ops config

# Display configuration with verbose output
quad-ops config --verbose

# Display configuration using custom config file
quad-ops config --config /path/to/config.yaml
```

## Sample Output

```yaml
repositorydir: /var/lib/quad-ops
syncinterval: 5m0s
quadletdir: /etc/containers/systemd
usermode: false
verbose: false
repositories:
  - url: https://github.com/user/repo.git
    branch: main
    path: containers
```

## Global Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config` | `-c` | Path to configuration file |
| `--verbose` | `-v` | Enable verbose output |

## Related Commands

- **[sync](sync)** - Synchronize repositories using the current configuration
- **[up](up)** - Start container units
- **[down](down)** - Stop container units
- **[unit](unit)** - Manage quadlet units

## See Also

- [Configuration](../configuration) - Detailed configuration options
- [Getting Started](../getting-started) - Initial setup guide
- [Repository Management](../repository-management) - Managing Git repositories