---
title: "Dependency Management"
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

# Dependency Management

Quad-Ops provides intelligent dependency management for your container services, ensuring proper start order and restart propagation.

## Service Dependencies

When you define a `depends_on` relationship in your Docker Compose file, Quad-Ops automatically:

1. Converts these relationships to systemd's `After` and `Requires` directives
2. Tracks reverse dependencies and converts them to `PartOf` relationships
3. Implements dependency-aware restart logic

### Example

In a Docker Compose file:

```yaml
services:
  db:
    image: docker.io/postgres:14

  app:
    image: docker.io/myapp:latest
    depends_on:
      - db

  web:
    image: docker.io/nginx:latest
    depends_on:
      - app
```

Quad-Ops converts this to systemd units with:

- `db.service`: No dependencies, but is `PartOf` app.service
- `app.service`: `After=db.service` and `Requires=db.service`, plus `PartOf` web.service
- `web.service`: `After=app.service` and `Requires=app.service`

## Dependency-Aware Restart Logic

Quad-Ops includes intelligent restart handling that:

1. Only restarts services with actual content changes
2. When multiple services change, only restarts the most foundational service
3. Propagates restarts up the dependency chain based on `PartOf` relationships

For example, if both `db` and `app` services change, Quad-Ops will:

1. Detect that `db` is a dependency of `app`
2. Only restart `db` directly
3. Allow systemd to automatically restart `app` and `web` due to dependency chain

This avoids unnecessary cascading restarts and ensures services come up in the correct order.

## Bidirectional Dependency Tree

Quad-Ops builds a bidirectional dependency tree to track:

- Dependencies: Services that a given service depends on
- Dependent Services: Services that depend on a given service

This complete dependency map enables intelligent restart decisions and ensures proper service ordering.

## Project Isolation

Dependencies are tracked within each project boundary, allowing separate projects to operate independently. Project names follow the format `<repo>-<folder>` to ensure proper isolation.