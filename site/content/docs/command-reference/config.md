---
title: "config"
weight: 50
---

# quad-ops config

Display the current configuration including defaults and overrides.

## Synopsis

```
quad-ops config [flags]
```

## Options

```
  -h, --help   help for config
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

The `config` command displays the current configuration for quad-ops, including all defaults and any overrides from configuration files or command-line flags.

This command is useful for:

- Verifying configuration settings
- Debugging configuration issues  
- Understanding active configuration values
- Checking default values and overrides

## Examples

### Display current configuration

```bash
quad-ops config
```

### Display configuration in JSON format

```bash
quad-ops config --output json
```

### Display configuration in YAML format

```bash
quad-ops config --output yaml
```

### Display configuration with verbose output

```bash
quad-ops config --verbose
```

### Display configuration using custom config file

```bash
quad-ops config --config /path/to/config.yaml
```

### Display user mode configuration

```bash
quad-ops config --user
```

## Sample Output

```yaml
repositorydir: /var/lib/quad-ops
quadletdir: /etc/containers/systemd
usermode: false
verbose: false
repositories:
  - name: quad-ops-compose
    url: https://github.com/trly/quad-ops-compose.git
    target: main
    cleanup:
      action: Delete
```

