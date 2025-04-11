---
title: "Secrets"
weight: 30
---

# Secrets

Secrets in Docker Compose are converted to Podman secret mounts within containers. Unlike Docker Swarm, Podman handles secrets as part of container configuration rather than as separate entities.

## Supported Properties

- `source`: Secret source name
- `target`: Mount path within container
- `uid`: User ID for the secret file
- `gid`: Group ID for the secret file
- `mode`: File permissions

## Example

```yaml
secrets:
  db_password:
    file: ./secrets/db_password.txt
  api_key:
    file: ./secrets/api_key.txt

services:
  web:
    image: nginx:latest
    secrets:
      - source: api_key
        target: /run/secrets/app_api_key
        uid: "1000"
        gid: "1000"
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