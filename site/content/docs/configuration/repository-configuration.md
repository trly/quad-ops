---
title: "Repository Configuration"
weight: 20
---

# Repository Configuration

Repository configuration defines how Quad-Ops manages individual Git repositories containing Docker Compose files.

## Repository Options

### Required Fields

| Option | Type | Description |
|--------|------|-------------|
| `name` | string | Unique identifier for the repository (used in unit naming) |
| `url` | string | Git repository URL (HTTPS, SSH, or file:// for local repos) |

### Optional Fields

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ref` | string | `main` | Git reference to checkout (branch, tag, or commit hash) |
| `composeDir` | string | `""` | Subdirectory containing Docker Compose files |
| `cleanup` | string | `"keep"` | Cleanup policy: `"keep"` or `"delete"` |
| `usePodmanDefaultNames` | boolean | `false` | Override global naming convention |

## Git Repository Sources

### HTTPS Repositories

```yaml
repositories:
  - name: public-app
    url: https://github.com/user/app.git
```

### SSH Repositories

```yaml
repositories:
  - name: private-app
    url: git@github.com:user/private-app.git
```

### Local Repositories

```yaml
repositories:
  - name: local-dev
    url: file:///home/user/my-project
```

## Git References

### Branch References

```yaml
repositories:
  - name: dev-app
    url: https://github.com/user/app.git
    ref: develop

  - name: staging-app
    url: https://github.com/user/app.git
    ref: staging
```

### Tag References

```yaml
repositories:
  - name: prod-app
    url: https://github.com/user/app.git
    ref: v2.1.0  # Specific version

  - name: latest-app
    url: https://github.com/user/app.git
    ref: latest  # Latest tag
```

### Commit Hash References

```yaml
repositories:
  - name: pinned-app
    url: https://github.com/user/app.git
    ref: abc123def456  # Specific commit
```

## Directory Structure

### Root-Level Compose Files

For repositories with Docker Compose files in the root:

```yaml
repositories:
  - name: simple-app
    url: https://github.com/user/simple-app.git
    # Looks for: docker-compose.yml, docker-compose.yaml, compose.yml, compose.yaml
```

### Subdirectory Compose Files

For repositories with compose files in subdirectories:

```yaml
repositories:
  - name: complex-app
    url: https://github.com/user/complex-app.git
    composeDir: deploy/docker
    # Looks in: deploy/docker/ for compose files
```

### Multiple Environment Structure

```yaml
repositories:
  - name: app-dev
    url: https://github.com/user/app.git
    composeDir: environments/dev

  - name: app-prod
    url: https://github.com/user/app.git
    composeDir: environments/prod
```

## Cleanup Policies

### Keep Policy (Default)

Units remain deployed even when removed from Docker Compose files:

```yaml
repositories:
  - name: persistent-app
    url: https://github.com/user/app.git
    cleanup: keep
```

**Use cases:**
- Production environments
- When manual control over unit lifecycle is desired
- Avoiding accidental service removal

### Delete Policy

Units are automatically removed when no longer in Docker Compose files:

```yaml
repositories:
  - name: dynamic-app
    url: https://github.com/user/app.git
    cleanup: delete
```

**Use cases:**
- Development environments
- Automated testing pipelines
- When repository changes should be fully reflected

## Naming Conventions

### Repository Names

Repository names become prefixes for all generated units:

```yaml
repositories:
  - name: myapp  # Creates units like: myapp-web.container, myapp-db.container
    url: https://github.com/user/myapp.git
```

**Best practices:**
- Use lowercase names
- Use hyphens instead of underscores
- Keep names concise but descriptive
- Avoid special characters

### Container Naming Examples

With `usePodmanDefaultNames: false` (default):
```yaml
# Repository: myapp, Service: web
# Container hostname: myapp-web
# Systemd unit: myapp-web.service
```

With `usePodmanDefaultNames: true`:
```yaml
# Repository: myapp, Service: web
# Container hostname: systemd-myapp-web
# Systemd unit: myapp-web.service
```

## Authentication

### SSH Key Authentication

For private repositories using SSH:

```bash
# Ensure SSH key is available
ssh-add ~/.ssh/id_rsa

# Test access
ssh -T git@github.com
```

### HTTPS with Tokens

For private HTTPS repositories, configure Git credentials:

```bash
# Store credentials (use with caution in production)
git config --global credential.helper store

# Or use environment variables
export GIT_USERNAME=token
export GIT_PASSWORD=ghp_your_token_here
```

## Advanced Examples

### Multi-Environment Setup

```yaml
repositories:
  # Development
  - name: app-dev
    url: https://github.com/company/app.git
    ref: develop
    composeDir: environments/dev
    cleanup: delete

  # Staging
  - name: app-staging
    url: https://github.com/company/app.git
    ref: staging
    composeDir: environments/staging
    cleanup: keep

  # Production
  - name: app-prod
    url: https://github.com/company/app.git
    ref: v2.1.0
    composeDir: environments/prod
    cleanup: keep
```

### Microservices Repository

```yaml
repositories:
  - name: auth-service
    url: https://github.com/company/microservices.git
    composeDir: services/auth

  - name: user-service
    url: https://github.com/company/microservices.git
    composeDir: services/user

  - name: api-gateway
    url: https://github.com/company/microservices.git
    composeDir: services/gateway
```

### Mixed Source Configuration

```yaml
repositories:
  # Public repository
  - name: opensource-tool
    url: https://github.com/project/tool.git
    ref: latest

  # Private repository
  - name: company-app
    url: git@github.com:company/private-app.git
    ref: main

  # Local development
  - name: local-dev
    url: file:///home/developer/workspace/project
    cleanup: delete
```

### Validation Commands

```bash
# Test repository access
git ls-remote https://github.com/user/repo.git

# Validate configuration
quad-ops config validate

# Check repository status
quad-ops unit list -t all
```

## Next Steps

- [Container Management](../container-management) - Learn how Quad-Ops processes Docker Compose files
- [Getting Started](../getting-started) - Set up your first repository