---
title: "Secrets"
---

# Secrets

Secrets in Docker Compose are converted to Podman secret mounts. This allows you to securely manage sensitive data in your containers.

## Supported Properties

- `source`: Secret source name (required)
- `target`: Mount path within container (defaults to /run/secrets/\<source\>)
- `uid`: User ID for the secret file (defaults to container's default user)
- `gid`: Group ID for the secret file (defaults to container's default group)
- `mode`: File permissions expressed as an octal number (defaults to "0644")

## Examples

### Basic File-Based Secret

```yaml
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

### Podman Environment Variable Secrets

```yaml
services:
  webapp:
    image: docker.io/myapp:latest
    secrets:
      - db_password
    # Podman-specific extension for environment variable secrets
    x-podman-env-secrets:
      db_password: DATABASE_PASSWORD  # Will be available as $DATABASE_PASSWORD in container

secrets:
  db_password:
    file: /path/to/secrets/db_password
```

### Using Both File and Environment Secrets

```yaml
services:
  webapp:
    image: docker.io/myapp:latest
    secrets:
      - source: db_password
        target: /run/secrets/db_password
      - api_key
    x-podman-env-secrets:
      db_password: DATABASE_PASSWORD  # Available both as file and environment variable
      api_key: API_KEY                # Available only as environment variable

secrets:
  db_password:
    file: /path/to/secrets/db_pass.txt
  api_key:
    file: /path/to/secrets/api_key.txt
```

## Conversion to Podman Secret Mounts

When Quad-Ops processes a secret definition from a Docker Compose file, it creates corresponding Podman secret mounts with the following mapping:

| Docker Compose Property | Podman Secret Property |
|-------------------------|------------------------|
| `source` | `Source` |
| `target` | `Target` |
| `uid` | `UID` |
| `gid` | `GID` |
| `mode` | `Mode` |

## Important Notes

1. **File-Based Secrets**: In Podman, secrets are implemented as direct file mounts from the host system into the container.

2. **Secret Files**: You must create the secret files on your host system before using them:
   ```bash
   mkdir -p /path/to/secrets
   echo "my-database-password" > /path/to/secrets/db_password
   chmod 600 /path/to/secrets/db_password
   ```

3. **Secret Paths**: Each secret is a file on the host that gets mounted read-only into the container at a specified location, typically under `/run/secrets/`.

4. **Default Permissions**: If the mode is not specified, it defaults to `0644`.

5. **Storage Location**: Store secrets outside repository directories in a secure location.

6. **Path Resolution**: Always use absolute paths for secret files to avoid resolution issues.

7. **Multiple Secrets**: You can define multiple secrets and assign different ones to different services:
   ```yaml
   secrets:
     db_password:
       file: ./secrets/db_password.txt
     api_key:
       file: ./secrets/api_key.txt
   ```