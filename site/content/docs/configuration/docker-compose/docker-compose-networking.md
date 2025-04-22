---
title: "Docker Compose Networking"
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

# Docker Compose Networking

## Container Hostname Resolution

Quad-Ops creates systemd-based DNS entries for your containers, allowing containers to communicate with each other by hostname.

### Hostname Pattern

By default, when containers are started by systemd through Podman Quadlet, they are automatically assigned a DNS name with the following pattern:

```
systemd-<unit-name>
```

In quad-ops, the unit name follows this structure:
```
<repository-name>-<directory>-<service-name>
```

Therefore, the complete DNS hostname becomes:
```
systemd-<repository-name>-<directory>-<service-name>
```

Where:
- `<repository-name>` is the name defined in your config.yaml
- `<directory>` is always the directory name containing the compose file
- `<service-name>` is the service name from your Docker Compose file

This `systemd-` prefix is automatically added by Podman's systemd integration as documented in the [podman-systemd.unit](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) documentation.

### Customizing Container Names

By default, quad-ops uses custom container names without the `systemd-` prefix, making container hostnames match exactly with their service names. This provides more predictable hostnames for your containers.

You can control whether containers use Podman's default naming scheme (with the `systemd-` prefix) or use the exact unit name without the prefix through the `usePodmanDefaultNames` setting in your configuration:

```yaml
# Global setting - applies to all repositories unless overridden
# Default is false: no systemd- prefix
usePodmanDefaultNames: false

repositories:
  - name: example-repo
    url: "https://github.com/example/repo.git"
    # Repository-specific override to use Podman's default naming
    usePodmanDefaultNames: true  # This repository will use Podman's default naming with systemd- prefix
```

When `usePodmanDefaultNames` is set to `true`, quad-ops will let Podman use its default behavior, which adds the `systemd-` prefix to all container hostnames.

### Example Hostname Resolution

Consider this configuration:

```yaml
repositories:
  - name: quad-ops
    url: "https://github.com/trly/quad-ops.git"
    ref: "main"
    composeDir: "examples"
    cleanup: "delete"
```

And a compose file at `examples/multi-service/docker-compose.yaml` with services `webapp` and `db`.

With the default configuration (`usePodmanDefaultNames: false`), the hostnames would be:
- `quad-ops-multi-service-webapp`
- `quad-ops-multi-service-db`

If you set `usePodmanDefaultNames: true`, the hostnames would be Podman's defaults:
- `systemd-quad-ops-multi-service-webapp`
- `systemd-quad-ops-multi-service-db`

In both cases, the hostname uses the actual directory containing the Docker Compose file (`multi-service`), regardless of the `composeDir` parameter.

### Network Aliases

To ensure compatibility with standard Docker Compose applications, quad-ops automatically adds the original service name as a network alias to each container. This allows containers to refer to each other using just the service name from the Docker Compose file, regardless of the actual container hostname.

For example, in a typical Docker Compose application:

```yaml
services:
  webapp:
    image: nginx
    depends_on:
      - db

  db:
    image: postgres
```

The `webapp` container can connect to the database using just `db` as the hostname, even though the actual container hostname might be `quad-ops-multi-service-db` or `systemd-quad-ops-multi-service-db` depending on your configuration.

This feature makes it easier to port existing Docker Compose applications to Podman without changing connection strings or environment variables.

### Best Practices

- Use the correct hostname format for your configuration when connecting services
  - With default settings (`usePodmanDefaultNames: false`): `<repository>-<directory>-<service>`
  - With `usePodmanDefaultNames: true`: `systemd-<repository>-<directory>-<service>`
  - For compatibility with standard Docker Compose: use just the service name (e.g., `db`)
- Test hostname resolution within containers using `ping <hostname>`
- For databases and other services that accept connection strings, make sure to use the correct hostname format
- Consider using environment variables to pass hostnames between containers
