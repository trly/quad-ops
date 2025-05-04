---
title: "Environment Files Example"
weight: 5
---

# Using Service-Specific Environment Files

This example demonstrates how to use service-specific environment files with Quad-Ops to properly handle environment variables containing special characters.

## Problem Statement

When working with containers, you may encounter environment variables that contain special characters like asterisks (`*`), spaces, quotes, or other characters that could be interpreted by the shell. These can cause issues during container startup.

For example, a variable like `ALLOWED_DOMAINS=*.example.com` might have the asterisk expanded by the shell, or a variable with spaces like `APP_NAME=My Web App` might be parsed incorrectly.

## Solution: Service-Specific Environment Files

Quad-Ops automatically detects and uses service-specific environment files in your Docker Compose directory. This allows Podman to handle the environment variables properly without shell interference.

## Example: Certificate Authority Container

Here's a complete example using Step CA, a certificate authority that needs environment variables with special characters:

**docker-compose.yml**:
```yaml
services:
  pki:
    image: docker.io/smallstep/step-ca:latest
    restart: always
    volumes:
      - certificate_authority:/home/step
    networks:
      - default
      - reverse-proxy
    secrets:
      - source: step_ca_password
        target: /home/step/secrets/password
        mode: 0400

volumes:
  certificate_authority:

secrets:
  step_ca_password:
    external: true

networks:
  default:
  reverse-proxy:
```

**pki.env**:
```env
# Environment variables for the PKI service
DOCKER_STEPCA_INIT_DNS_NAMES=*.quad-ops.local
DOCKER_STEPCA_INIT_NAME=Quad-Ops Test CA
```

## How It Works

1. Save the Docker Compose file as shown above
2. Create a file named `pki.env` in the same directory
3. Add the environment variables to the `.env` file
4. When Quad-Ops processes the Docker Compose file, it will:
   - Detect the `pki.env` file
   - Add it to the container unit as an `EnvironmentFile` directive
   - Podman will read the environment variables from the file, handling special characters correctly

## Generated Quadlet Unit

The generated `home-infrastructure-pki.container` file will look like this:

```
[Unit]
Description=PKI Certificate Authority
After=home-infrastructure-reverse-proxy-network.service
Requires=home-infrastructure-reverse-proxy-network.service

[Container]
Image=docker.io/smallstep/step-ca:latest
Label=managed-by=quad-ops
EnvironmentFile=/path/to/home-infrastructure/pki.env
Volume=home-infrastructure-certificate_authority.volume:/home/step
Network=home-infrastructure-default.network
Network=home-infrastructure-reverse-proxy.network
NetworkAlias=pki
ContainerName=home-infrastructure-pki
Secret=step_ca_password,mode=0400,target=/home/step/secrets/password

[Service]
Restart=always

[Install]
WantedBy=default.target
```

## Benefits

- Special characters in environment variables are handled correctly
- Credentials and sensitive values can be separated from Docker Compose files
- Multiple environment files can be used for different deployment contexts
- More readable and maintainable configuration
- Follows Podman Quadlet best practices