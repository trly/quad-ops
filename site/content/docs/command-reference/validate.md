---
title: "validate"
weight: 70
---

# quad-ops validate

Validates Docker Compose files and quad-ops extensions in a repository, directory, or single file.

Can clone a git repository and validate all Docker Compose files within it, validate all
compose files in a local directory, or validate a single compose file. Perfect for CI/CD
pipelines and development workflows. The validation checks for:

- Valid Docker Compose file syntax
- Quad-ops extension compatibility
- Security requirements for secrets and environment variables
- Service dependency graph integrity
- Build configuration validity

Examples:

# Validate files in current directory

  quad-ops validate

# Validate files in specific directory

  quad-ops validate /path/to/compose/files

# Validate a single compose file (great for CI)

  quad-ops validate docker-compose.yml
  quad-ops validate /path/to/my-service.compose.yml

# Clone and validate a git repository (use --repo flag, NOT path argument)

  quad-ops validate --repo <https://github.com/user/repo.git>

# Clone specific branch/tag and validate

  quad-ops validate --repo <https://github.com/user/repo.git> --ref main

# Validate specific compose directory in repository

  quad-ops validate --repo <https://github.com/user/repo.git> --compose-dir services

Note: Use either a local path OR the --repo flag, but not both.

## Synopsis

```
quad-ops validate [path] [flags]
```

## Options

```
      --check-system         Check system requirements (systemd, podman) before validation
      --compose-dir string   Subdirectory within repository containing compose files
  -h, --help                 help for validate
      --ref string           Git reference (branch/tag/commit) to checkout (default "main")
      --repo string          Git repository URL to clone and validate
      --skip-clone           Skip cloning if repository already exists locally
      --temp-dir string      Custom temporary directory for cloning (default: system temp)
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
