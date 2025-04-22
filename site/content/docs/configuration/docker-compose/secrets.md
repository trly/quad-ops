---
title: "Secrets"
---

# Using Secrets with Quad-Ops

Quad-Ops supports Docker Compose secrets for managing sensitive data in your containers. Unlike Docker Swarm, Podman implements secrets as direct file mounts, which makes them simpler to work with but requires some specific configuration.

## How Podman Secrets Work

In Podman, secrets are implemented as direct file mounts from the host system into the container. Each secret is a file on the host that gets mounted read-only into the container at a specified location, typically under `/run/secrets/`.

Quad-Ops translates Docker Compose secret definitions into the appropriate Podman Quadlet format.

## Supported Properties

- **source**: Secret source name (required)
- **target**: Mount path within container (defaults to /run/secrets/<source>)
- **uid**: User ID for the secret file (defaults to container's default user)
- **gid**: Group ID for the secret file (defaults to container's default group)
- **mode**: File permissions expressed as an octal number (defaults to "0644")

## Creating Secrets

Before using secrets in your Docker Compose file, you must create the secret files on your host system:

```bash
# Create a directory for your secrets (if it doesn't exist)
mkdir -p /path/to/secrets

# Create a secret file
echo "my-database-password" > /path/to/secrets/db_password

# Secure the secret file
chmod 600 /path/to/secrets/db_password
```

## Example Configuration

### Basic Example

```yaml
version: '3.9'

services:
  webapp:
    image: docker.io/myapp:latest
    secrets:
      - source: db_password
        target: /run/secrets/db_password
        mode: 0400
        uid: "1000"
        gid: "1000"

secrets:
  db_password:
    file: /path/to/secrets/db_password
```

### Multiple Secrets Example

```yaml
secrets:
  db_password:
    file: ./secrets/db_password.txt
  api_key:
    file: ./secrets/api_key.txt
  ssl_cert:
    file: ./secrets/ssl_cert.pem

services:
  web:
    image: nginx:latest
    secrets:
      - source: api_key
        target: /run/secrets/app_api_key
        uid: "1000"
        gid: "1000"
        mode: 0400
      - source: ssl_cert
        target: /run/secrets/server.cert
        mode: 0400

  db:
    image: postgres:latest
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    secrets:
      - db_password
```

## Secret Handling in Podman

When Quad-Ops processes secrets from a Docker Compose file, it adds them to the container configuration. In Podman, secrets are handled as files mounted into the container, usually in the `/run/secrets/` directory.

The secret properties are mapped as follows:

| Docker Compose Property | Podman Secret Property |
|-------------------------|------------------------|
| `source` | `Source` |
| `target` | `Target` |
| `uid` | `UID` |
| `gid` | `GID` |
| `mode` | `Mode` |

If the mode is not specified, it defaults to `0644`.

## Best Practices for Podman Secrets

1. **Store secrets outside repository directories**: Keep secret files in a secure location outside your Git repositories.

2. **Use restrictive permissions**: Set appropriate file permissions (600 or 400) for your secret files.

3. **Use absolute paths**: Always use absolute paths for secret files to avoid resolution issues.

4. **Consistent naming**: Use consistent naming between the secret definition and reference in the service.

5. **Document secrets**: Document which secrets are needed but don't include the actual secret content.

6. **Use secret rotation**: Implement a process for rotating secrets regularly.

7. **Validate secret existence**: Ensure all required secret files exist before deploying.

## Limitations

Compared to Docker Swarm's secrets management, Podman secrets have some limitations:

1. **No built-in encryption**: Secrets are stored as plain files on the host.

2. **No centralized management**: Each secret must be manually created on each host.

3. **File-based only**: All secrets must exist as files on the host system.

4. **No automatic rotation**: Secrets don't have built-in rotation mechanisms.

5. **Host dependency**: Secret files must be managed on the host outside of the container lifecycle.

## Using Secrets in Your Application

Your application can read the secret directly from the mounted file:

```bash
# Inside the container
cat /run/secrets/db_password
```