---
title: "Environment Variable Secrets"
weight: 40
---

# Environment Variable Secrets

Quad-Ops supports using Podman's `secret type=env` feature to provide secrets as environment variables, using Docker Compose extensions.

## Overview

In standard Docker Compose, secrets are only available as files inside the container. However, Podman Quadlet's systemd units support exposing secrets as environment variables using the `type=env` option.

Quad-Ops allows you to use this feature by adding a Docker Compose extension called `x-podman-env-secrets` that maps secret names to environment variable names.

## Example

```yaml
services:
  app:
    image: docker.io/myapp
    secrets:
      - db_password
      - api_key
    x-podman-env-secrets:
      db_password: DB_PASSWORD  # Will be available as $DB_PASSWORD
      api_key: API_KEY          # Will be available as $API_KEY

secrets:
  db_password:
    file: ./secrets/db_password.txt
  api_key:
    file: ./secrets/api_key.txt
```

## Using External Secrets

You can also use external secrets (secrets that exist outside your Docker Compose project) with environment variable mapping:

```yaml
services:
  app:
    image: docker.io/myapp
    secrets:
      - db_password
      - jwt_key
    x-podman-env-secrets:
      db_password: DATABASE_PASSWORD
      jwt_key: JWT_SECRET_KEY

secrets:
  db_password:
    external: true  # Secret exists on the host system
  jwt_key:
    external: true
```

With external secrets, the secret must already exist on the host system at one of these locations:

- System-level secrets: `/run/secrets/<secret-name>`
- User-level secrets: `$XDG_RUNTIME_DIR/secrets/<secret-name>` (typically `/run/user/<uid>/secrets/<secret-name>`)

## How It Works

When processing a service with the `x-podman-env-secrets` extension, Quad-Ops will:

1. Generate the standard file-based secret entries (as usual)
2. Generate additional `type=env` secret directives for any secrets specified in the extension

The generated systemd unit file will include lines like:

```ini
[Container]
# Other container settings...
Secret=db_password,type=env,target=DB_PASSWORD
Secret=api_key,type=env,target=API_KEY
```

## Security Considerations

When using environment variable secrets:

1. Secret values will be visible in the container environment
2. Use caution with containers that dump environment variables (like some debug tools)
3. Consider file-based secrets for highly sensitive values
4. For external secrets, ensure they exist in the proper Podman secret locations before container startup

## Implementation Notes

- The secrets must be defined in the standard secrets section of the Docker Compose file
- The `x-podman-env-secrets` extension maps secret names to environment variable names
- Both file-based and environment-based versions of the same secret can be used simultaneously
- Works with both local secret files and external secrets