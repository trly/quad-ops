---
title: "Build"
---

# Build Configuration

Quad-Ops supports Docker Compose build configurations, converting them to Podman Quadlet build units. This allows for images to be built automatically as part of the deployment process.

## Build Support Overview

When a service in your Docker Compose file includes a `build` directive, Quad-Ops creates a corresponding Podman Quadlet build unit. This build unit is responsible for building the container image that will be used by the service.

Build units are created in addition to the container units and are executed before the container starts, ensuring that the latest image is always available.

## Supported Properties

The following build properties are supported:

- `context`: Build context path (directory containing the Dockerfile)
- `dockerfile`: Path to the Dockerfile/Containerfile (relative to context)
- `args`: Build arguments for the build process
- `labels`: Labels to apply to the resulting image
- `target`: Specify the build target stage
- `network`: Network mode for build (host, none, or a project-defined network)
- `pull`: Whether to always pull base images (true/false)
- `secrets`: Secrets to expose to the build process

## Advanced Build Features

Quad-Ops also supports several advanced build features through extensions:

- **SSH Authentication**: Allows secure authentication during build
- **Cache Configuration**: Controls build cache behavior
- **Volume Mounts**: Mounts volumes during the build process
- **Custom Build Arguments**: Passes additional arguments to the build process

### Extension Properties

For features not directly supported by Podman Quadlet build units, Quad-Ops provides extensions:

- `x-podman-volumes`: Mount volumes during build
- `x-podman-buildargs`: Pass additional arguments to podman build

## Example

```yaml
services:
  webapp:
    image: webapp:latest
    build:
      context: ./app
      dockerfile: Dockerfile.prod
      args:
        VERSION: "1.0"
        BUILD_DATE: "2025-05-22"
      labels:
        org.opencontainers.image.source: "https://github.com/example/webapp"
      target: production
      network: host
      pull: true
      secrets:
        - source: npm_token
          target: NPM_TOKEN
    # Other service configuration...

  api:
    image: api:latest
    build:
      context: ./api
      dockerfile: Dockerfile
      args:
        DEBUG: "false"
      # Advanced features using extensions
      x-podman-volumes:
        - "./cache:/root/.cache"
      x-podman-buildargs:
        - "--ssh=default"
        - "--cache-from=api:cache"
    # Other service configuration...

secrets:
  npm_token:
    file: ./secrets/npm_token.txt
```

## Conversion to Podman Build Units

When Quad-Ops processes a build configuration from a Docker Compose file, it creates a corresponding Podman build unit with the following mapping:

| Docker Compose Property | Podman Build Property |
|-------------------------|----------------------|
| `build.context` | `SetWorkingDirectory` |
| `build.dockerfile` | `File` |
| `build.args` | `Environment` (for build args) |
| `build.labels` | `Label` |
| `build.target` | `Target` |
| `build.network` | `Network` |
| `build.pull` | `Pull` |
| `build.secrets` | `Secret` |
| `image` | `ImageTag` |
| `x-podman-volumes` | `Volume` |
| `x-podman-buildargs` | `PodmanArgs` |

## Build Unit Naming

Build units follow a specific naming convention:

- Build unit file: `<project-name>-<service-name>-build.build`
- Systemd service: `<project-name>-<service-name>-build.service`

## Important Notes

1. **Git URLs**: Build contexts can be Git URLs (starting with http:// or https://), which will be cloned during the build process.

2. **SSH Authentication**: SSH authentication for Git operations during build is supported via the `x-podman-buildargs` extension with `--ssh=default`.

3. **Build Cache**: Build cache settings are supported via the `x-podman-buildargs` extension with options like `--cache-from`.

4. **Default Image Tag**: If no image is specified, a default image tag is generated based on the service name (e.g., `localhost/service-name:latest`).

5. **Build Dependencies**: Build units are automatically configured as dependencies for their corresponding container units.

6. **Working Directory**: For local build contexts, the build context path is resolved relative to the repository's working directory.

7. **Pull Policy**: The default pull policy is `missing`, which only pulls base images if they don't exist locally.

8. **Volume Mounts**: Volume mounts for the build process can be specified using the `x-podman-volumes` extension, which is particularly useful for caching dependencies between builds.