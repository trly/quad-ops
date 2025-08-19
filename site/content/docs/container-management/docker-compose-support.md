---
title: "Docker Compose Support"
weight: 20
---

# Docker Compose Support

Quad-Ops converts Docker Compose files to Podman Quadlet units for systemd management. For comprehensive documentation on Docker Compose syntax and features, see the [Compose Specification](https://compose-spec.io/).

## Supported Compose Versions

- **No version specified** (treated as 3.x) **[Recommended]**
- **Version 3.0** through **3.8** (latest)
- **Version 2.x** (partial compatibility)

## Standard Docker Compose Features

Quad-Ops supports the full Docker Compose specification. For detailed documentation on standard features, refer to:

- [Services](https://compose-spec.io/spec/#services-top-level-element) - Container configuration
- [Networks](https://compose-spec.io/spec/#networks-top-level-element) - Network definitions
- [Volumes](https://compose-spec.io/spec/#volumes-top-level-element) - Volume management
- [Build](https://compose-spec.io/spec/#build) - Image building
- [Deploy](https://compose-spec.io/spec/#deploy) - Resource constraints and deployment
- [Healthcheck](https://compose-spec.io/spec/#healthcheck) - Container health monitoring

## Podman-Specific Extensions

### Environment Secrets

Map Podman secrets to environment variables:

```yaml
services:
  app:
    environment:
      - DB_PASSWORD_FILE=/run/secrets/db_password
    x-podman-env-secrets:
      DB_PASSWORD: db_password  # secret name -> env var
      API_KEY: api_secret
```

### Volume Extensions

Podman-specific volume options:

```yaml
services:
  app:
    volumes:
      - "data:/data"
    x-podman-volumes:
      - "cache:/tmp/cache:O"  # Overlay mount
      - "logs:/logs:U"        # Chown to container user
```

### Build Extensions

Additional build arguments:

```yaml
services:
  app:
    build:
      context: .
    x-podman-buildargs:
      BUILDKIT_INLINE_CACHE: "1"
      BUILDPLATFORM: "linux/amd64"
```

## Conversion Examples

### Docker Compose to Quadlet

**Docker Compose:**
```yaml
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html
    depends_on:
      - app

  app:
    build: .
    environment:
      - NODE_ENV=production
```

**Generated Quadlet Units:**

`myproject-web.container`:
```ini
[Unit]
Description=myproject-web container
After=myproject-app.service

[Container]
Image=docker.io/library/nginx:latest
PublishPort=8080:80
Volume=./html:/usr/share/nginx/html
NetworkAlias=web

[Service]
Restart=always

[Install]
WantedBy=default.target
```

`myproject-app.container`:
```ini
[Unit]
Description=myproject-app container

[Container]
Image=localhost/myproject-app:latest
Environment=NODE_ENV=production
NetworkAlias=app

[Service]
Restart=always

[Install]
WantedBy=default.target
```

## Quad-Ops Validation

Validate Docker Compose files before deployment:

```bash
# Validate compose files with quad-ops extensions
quad-ops validate docker-compose.yml

# Validate all compose files in directory
quad-ops validate ./compose-files/

# Validate remote repository
quad-ops validate --repo https://github.com/user/repo.git

# Test compose conversion without applying
quad-ops sync --dry-run

# Check generated Quadlet units
ls /etc/containers/systemd/
```

The `validate` command checks for:
- Docker Compose syntax and structure
- Quad-ops extension compatibility  
- Security requirements (secrets, env vars)
- DNS naming conventions
- File path security

## Next Steps

- [Environment Files](environment-files) - Environment variable management
- [Build Support](build-support) - Docker build configurations
- [Supported Features](../podman-systemd/supported-features) - Feature compatibility matrix
- [Compose Specification](https://compose-spec.io/) - Official Docker Compose documentation