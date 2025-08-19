---
title: "validate"
description: "Validates Docker Compose files and quad-ops extensions in repositories, directories, or single files"
lead: "The validate command provides comprehensive validation of Docker Compose files with quad-ops extensions, perfect for CI/CD pipelines and development workflows."
date: 2025-01-18T00:00:00+00:00
lastmod: 2025-01-18T00:00:00+00:00
draft: false
images: []
menu:
  docs:
    parent: "command-reference"
weight: 150
toc: true
---

## Overview

The `validate` command validates Docker Compose files and quad-ops extensions in repositories, directories, or single files. It's designed to catch configuration errors early and ensure compatibility with quad-ops before deployment.

## Usage

```bash
quad-ops validate [path] [flags]
```

## Arguments

- `path` (optional): Path to validate. Can be:
  - A directory containing Docker Compose files
  - A single Docker Compose file (`.yml` or `.yaml`)
  - Omitted to validate current directory

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--repo` | | Git repository URL to clone and validate | |
| `--ref` | | Git reference (branch/tag/commit) to checkout | `main` |
| `--compose-dir` | | Subdirectory within repository containing compose files | |
| `--skip-clone` | | Skip cloning if repository already exists locally | `false` |
| `--temp-dir` | | Custom temporary directory for cloning | system temp |
| `--check-system` | | Check system requirements (systemd, podman) before validation | `false` |

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
- **Sensitive data**: Detects potentially insecure test/default values

### Quad-ops Extensions
- **Init containers**: Validates quad-ops init container configurations
- **Build configurations**: Checks custom build settings and contexts
- **Dependency relationships**: Validates service dependency graphs

## Examples

### Directory Validation

Validate all compose files in the current directory:
```bash
quad-ops validate
```

Validate compose files in a specific directory:
```bash
quad-ops validate /path/to/compose/files
```

### Single File Validation (CI/CD)

Perfect for continuous integration pipelines:

```bash
# Validate a standard compose file
quad-ops validate docker-compose.yml

# Validate custom named compose file
quad-ops validate my-service.yml

# Validate nested compose file
quad-ops validate compose/production.yaml
```

### Repository Validation

Clone and validate a git repository:
```bash
quad-ops validate --repo https://github.com/user/repo.git
```

Validate specific branch or tag:
```bash
quad-ops validate --repo https://github.com/user/repo.git --ref v1.2.3
```

Validate specific directory within repository:
```bash
quad-ops validate --repo https://github.com/user/repo.git --compose-dir services
```

### CI/CD Integration

The validate command is designed for seamless integration with CI/CD pipelines:

#### GitHub Actions
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

#### GitLab CI
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

#### Jenkins Pipeline
```groovy
pipeline {
    agent any
    stages {
        stage('Validate Compose') {
            steps {
                sh '''
                    curl -sSL https://github.com/trly/quad-ops/raw/main/install.sh | bash
                    quad-ops validate docker-compose.yml
                '''
            }
        }
    }
}
```

## Advanced Usage

### Multiple File Validation

Validate multiple compose files using shell globbing:
```bash
# Validate all yml files in compose directory
for file in compose/*.yml; do
    quad-ops validate "$file"
done

# Or use find command
find . -name "*.yml" -exec quad-ops validate {} \;
```

### Conditional Validation

Use with conditional logic for complex workflows:
```bash
# Only deploy if validation passes
quad-ops validate docker-compose.yml && docker-compose up -d
```

### Verbose Output

Enable verbose logging for debugging:
```bash
quad-ops -v validate docker-compose.yml
```

## Output and Exit Codes

### Success Output
```
✓ All 1 projects validated successfully
```

### Validation Errors
```
Validation Summary:
✓ Valid projects: 0
✗ Invalid projects: 1

Validation Errors:
  • Project myapp: service webapp: invalid secret reference db_password: 
    secret name must follow DNS naming conventions: db_password
```

### Exit Codes
- **0**: All validations passed
- **1**: Validation errors found or command failed

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

### Repository Cloning Issues
If repository cloning fails:
- Check that the repository URL is accessible
- Verify the specified reference (branch/tag) exists
- Ensure you have necessary authentication for private repositories
- Use `--temp-dir` to specify a custom temporary directory

### System Requirements
By default, validate doesn't check system requirements to allow usage on systems without systemd/podman:
- Use `--check-system` if you want to verify system requirements
- Validation focuses on file content, not runtime environment

## Related Commands

- [`quad-ops sync`]({{< relref "sync" >}}) - Synchronize and deploy validated compose files
- [`quad-ops unit list`]({{< relref "unit/list" >}}) - List generated units after sync
- [`quad-ops up`]({{< relref "up" >}}) - Start services after validation and sync
