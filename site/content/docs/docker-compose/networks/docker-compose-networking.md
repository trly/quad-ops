---
title: "Container Networking"
---

# Container Networking

## Hostname Resolution

Quad-Ops creates DNS entries for your containers, allowing them to communicate with each other by hostname.

### Default Naming Behavior

By default, Quad-Ops configures containers with hostnames that match their service names **without** the `systemd-` prefix that Podman normally adds. This makes DNS resolution more predictable and similar to Docker Compose.

The unit and hostname structure follows:
```
<repository-name>-<directory>-<service-name>
```

Where:
- `<repository-name>` is the name defined in your config.yaml
- `<directory>` is the directory name containing the compose file
- `<service-name>` is the service name from your Docker Compose file

### Customizing Container Names

You can control the hostname naming through the `usePodmanDefaultNames` setting:

```yaml
# Global setting (default: false - no systemd- prefix)
usePodmanDefaultNames: false

repositories:
  - name: example-repo
    url: "https://github.com/example/repo.git"
    # Repository-specific override
    usePodmanDefaultNames: true  # Uses systemd- prefix for this repo only
```

### Example

For a Docker Compose file with services `webapp` and `db`:

| Configuration | Resulting Hostnames |
|---------------|---------------------|
| `usePodmanDefaultNames: false` (default) | `repo-dir-webapp`<br>`repo-dir-db` |
| `usePodmanDefaultNames: true` | `systemd-repo-dir-webapp`<br>`systemd-repo-dir-db` |

### Service Name Resolution

For compatibility with standard Docker Compose, Quad-Ops automatically adds the service name as a network alias to each container. This allows containers to refer to each other using just the service name from the Docker Compose file.

For example, in this Docker Compose application:

```yaml
services:
  webapp:
    image: docker.io/nginx:latest
    depends_on:
      - db
    environment:
      - DB_HOST=db  # Works regardless of actual container hostname

  db:
    image: docker.io/postgres:latest
```

The `webapp` container can connect to the database using just `db` as the hostname, even though the actual container hostname may be different.
