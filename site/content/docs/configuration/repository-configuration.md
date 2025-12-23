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
| `ref` | string | remote HEAD | Git reference to checkout (branch, tag, or commit hash) |
| `composeDir` | string | `""` | Subdirectory containing Docker Compose files |

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

Quad-Ops **recursively** scans for compose files from the scan root. The scan root is the repository root when `composeDir` is not set, or the specified subdirectory when it is. All compose files found anywhere in the directory tree are loaded as separate projects.

Recognized compose file names (in priority order): `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`.

### Root-Level Compose Files

For repositories with Docker Compose files in the root:

```yaml
repositories:
  - name: simple-app
    url: https://github.com/user/simple-app.git
    # Recursively scans the repository root for compose files
```

### Subdirectory Compose Files

For repositories with compose files in subdirectories:

```yaml
repositories:
  - name: complex-app
    url: https://github.com/user/complex-app.git
    composeDir: deploy/docker
    # Recursively scans deploy/docker/ for compose files
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

## Naming Conventions

### Unit Name Prefixes

Generated unit names use the pattern `{project}-{service}.container`, where the **project name is the directory containing the compose file** — not the repository name.

When compose files are at the repository root, the project name matches the repository name (since the repository is cloned into a directory named after `name`):

```yaml
repositories:
  - name: myapp  # Cloned to .../myapp/, project name = "myapp"
    url: https://github.com/user/myapp.git
    # Creates units like: myapp-web.container, myapp-db.container
```

When `composeDir` is set, the project name is the **last path component** of the compose file's directory:

```yaml
repositories:
  - name: app-dev
    url: https://github.com/user/app.git
    composeDir: environments/dev
    # Project name = "dev", creates units like: dev-web.container
```

**Best practices for repository names:**
- Use lowercase names
- Use hyphens instead of underscores
- Keep names concise but descriptive
- Avoid special characters

### Systemd Unit Examples

```yaml
# Repository: myapp (no composeDir), Service: web
# Quadlet file: myapp-web.container
# Systemd unit: myapp-web.service
```

## Authentication

Quad-Ops uses [go-git](https://github.com/go-git/go-git) for repository operations, which has different authentication behavior than the native `git` CLI. Notably, go-git does **not** support Git credential helpers, `.netrc` files, or environment variables like `GIT_USERNAME`.

### SSH Key Authentication

SSH repositories require a running SSH agent with keys loaded. go-git connects to the agent via the `SSH_AUTH_SOCK` environment variable — it does **not** read key files from `~/.ssh/` directly.

```bash
# Start the SSH agent (if not already running)
eval "$(ssh-agent -s)"

# Add your key to the agent
ssh-add ~/.ssh/id_ed25519

# Verify the agent has your key
ssh-add -l

# Test access
ssh -T git@github.com
```

{{< hint warning >}}
If `SSH_AUTH_SOCK` is not set or no agent is running, SSH clones will fail. When running Quad-Ops as a systemd service, ensure the agent socket is available in the service environment.
{{< /hint >}}

### HTTPS Repositories

For **public** HTTPS repositories, no authentication is needed.

For **private** HTTPS repositories, go-git only supports credentials embedded in the URL:

```yaml
repositories:
  - name: private-app
    url: https://username:token@github.com/user/private-app.git
```

{{< hint danger >}}
Embedding credentials in URLs is stored in your configuration file. Ensure the file has appropriate permissions (`chmod 600`) and never commit it to version control.
{{< /hint >}}

### Host Key Verification

For SSH repositories, go-git verifies host keys against known hosts files in this order:

1. Files listed in the `SSH_KNOWN_HOSTS` environment variable
2. `~/.ssh/known_hosts`
3. `/etc/ssh/ssh_known_hosts`

If the remote host is not found in any of these files, the connection will fail. Add the host key before first use:

```bash
ssh-keyscan github.com >> ~/.ssh/known_hosts
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

  # Staging
  - name: app-staging
    url: https://github.com/company/app.git
    ref: staging
    composeDir: environments/staging

  # Production
  - name: app-prod
    url: https://github.com/company/app.git
    ref: v2.1.0
    composeDir: environments/prod
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
```

### Validation Commands

```bash
# Test repository access
git ls-remote https://github.com/user/repo.git

# Validate compose files
quad-ops validate
```

## Next Steps

- [Quick Start](../quick-start/) - Set up your first repository
- [Command Reference](../command-reference/) - Available commands and options
