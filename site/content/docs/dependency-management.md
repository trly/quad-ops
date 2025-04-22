---
title: "Dependency Management"
---

# Dependency Management

Quad-Ops provides intelligent dependency management for container services, networks, and volumes, ensuring proper start order and restart propagation.

## Types of Dependencies

Quad-Ops automatically manages three types of dependencies:

1. **Service Dependencies** - Defined by `depends_on` in Docker Compose
2. **Network Dependencies** - Containers depend on networks they connect to
3. **Volume Dependencies** - Containers depend on volumes they mount

## Systemd Integration

When you define a `depends_on` relationship in your Docker Compose file, Quad-Ops converts these to systemd directives:

| Docker Compose Relationship | Systemd Directive | Purpose |
|-----------------------------|-------------------|-------------------|
| `depends_on` | `After=` and `Requires=` | Ensures dependent starts after prerequisite |
| Reverse dependency | `PartOf=` | When prerequisite restarts, dependent also restarts |

### Example: Service Dependencies

```yaml
services:
  db:
    image: docker.io/library/postgres:14

  app:
    image: docker.io/myapp:latest
    depends_on:
      - db

  web:
    image: docker.io/library/nginx:latest
    depends_on:
      - app
```

Quad-Ops creates these systemd relationships:

- `db.service`: No dependencies, but is `PartOf` app.service
- `app.service`: `After=db.service` and `Requires=db.service`, plus `PartOf` web.service
- `web.service`: `After=app.service` and `Requires=app.service`

## Intelligent Restart Handling

Quad-Ops implements dependency-aware restart logic to minimize service disruption:

- Only restarts services with actual content changes
- When multiple services change, only restarts the most foundational service
- Uses systemd's `PartOf` to propagate restarts up the dependency chain
- Infrastructure (networks/volumes) is always started before containers

### Example: Multiple Service Changes

If both `db` and `app` services change, Quad-Ops will:

1. Detect that `db` is a dependency of `app`
2. Only restart `db` directly
3. systemd automatically restarts `app` and `web` due to dependency chain

This avoids unnecessary cascading restarts while ensuring proper service ordering.

## Project Isolation

Dependencies are tracked within each project boundary, allowing separate projects to operate independently. Project names follow the format `<repo>-<folder>` to ensure proper isolation.