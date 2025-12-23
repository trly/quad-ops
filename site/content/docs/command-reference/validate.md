---
title: "validate"
weight: 70
---

# quad-ops validate

Validates Docker Compose files and quad-ops extensions.

When a path is provided, validates the specified file or all compose files in the directory.
When no path is provided, validates all compose files from repositories defined in the configuration.

## Synopsis

```
quad-ops validate [path]
```

## Arguments

```
  [path]   Optional path to a compose file or directory to validate
```

## Global Options

```
    --config string   Path to the configuration file
    --debug           Enable debug mode
    --verbose         Enable verbose output
```

## Validation Checks

The validate command performs comprehensive checks on your Docker Compose files:

### Core Validation

- **Docker Compose syntax**: Validates YAML structure and compose specification compliance
- **Service configuration**: Checks service definitions, images, and networking
- **Resource definitions**: Validates volumes, networks, and secrets

### Security Validation

- **Environment variables**: Validates variable names follow POSIX conventions
- **Secret validation**: Ensures secret names follow DNS naming conventions
- **File paths**: Validates secret file paths are absolute and secure

### Quad-ops Extensions

- **Build configurations**: Checks custom build settings and contexts
- **Dependency relationships**: Validates service dependency graphs

## Examples

### Validate all repositories from configuration

```bash
quad-ops validate
```

### Validate a single compose file

```bash
quad-ops validate docker-compose.yml
```

### Validate all compose files in a directory

```bash
quad-ops validate /path/to/compose/files
```

### Validate with verbose output

```bash
quad-ops validate --verbose docker-compose.yml
```

## CI/CD Integration

The validate command is designed for seamless integration with CI/CD pipelines:

### GitHub Actions

```yaml
name: Validate Compose Files
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install quad-ops
        run: |
          curl -sSL https://github.com/trly/quad-ops/raw/main/install.sh | bash
      - name: Validate compose files
        run: |
          quad-ops validate docker-compose.yml
```

### GitLab CI

```yaml
validate-compose:
  stage: validate
  image: ubuntu:latest
  before_script:
    - apt-get update && apt-get install -y curl
    - curl -sSL https://github.com/trly/quad-ops/raw/main/install.sh | bash
  script:
    - quad-ops validate docker-compose.yml
  only:
    changes:
      - "*.yml"
      - "*.yaml"
```

## Common Validation Issues

### DNS Naming Violations

Secret names must follow DNS naming conventions (no underscores):

❌ **Invalid:**

```yaml
secrets:
  db_password:  # Contains underscore
    file: ./password.txt
```

✅ **Valid:**

```yaml
secrets:
  db-password:  # Uses hyphen
    file: ./password.txt
```

### Environment Variable Issues

Environment variable keys must follow POSIX naming:

❌ **Invalid:**

```yaml
environment:
  123VAR: value     # Starts with number
  my-var: value     # Contains hyphen
```

✅ **Valid:**

```yaml
environment:
  MY_VAR: value     # Alphanumeric + underscore
  VAR123: value     # Can end with number
```

### Secret File Paths

Secret file paths must be absolute:

❌ **Invalid:**

```yaml
secrets:
  my-secret:
    file: ./secret.txt  # Relative path
```

✅ **Valid:**

```yaml
secrets:
  my-secret:
    file: /path/to/secret.txt  # Absolute path
```

## Troubleshooting

### File Not Recognized

If a YAML file isn't recognized as a compose file:

- Ensure the file has a `.yml` or `.yaml` extension
- Check that the file contains valid YAML syntax
- Verify the file has a `services:` section
