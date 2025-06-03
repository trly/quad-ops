# Init Containers Example

This example demonstrates the use of the `x-quad-ops-init` extension to run initialization containers before the main services start.

## Overview

The `x-quad-ops-init` extension allows you to define one or more init containers that will run before the main container starts, similar to Kubernetes init containers.

## Extension Format

```yaml
x-quad-ops-init:
  - image: docker.io/example/image:tag
    command: run this command
  - image: another/image:tag
    command: ["sh", "-c", "multi word command"]
```

### Fields

- `image` (required): The container image to run for initialization
- `command` (optional): The command to run in the init container. Can be a string or array of strings.

## How It Works

1. **Init Container Generation**: Each entry in `x-quad-ops-init` generates a separate Quadlet unit with:
   - Name pattern: `<project>-<service>-init-<index>`
   - Service type: `oneshot`
   - `RemainAfterExit=yes` to keep the service in a "started" state

2. **Dependency Management**: The main service container automatically depends on all init containers:
   - `After=<init-container>.service`
   - `Requires=<init-container>.service`

3. **Execution Order**: Init containers run sequentially in the order defined, and the main container only starts after all init containers complete successfully.

## Example Usage

In this example:

### Web Service
- Runs two init containers before starting nginx
- First init container: Simple logging and delay
- Second init container: Sets up shared data directory

### Database Service  
- Runs one init container before starting PostgreSQL
- Init container: Simulates database migration

## Running the Example

```bash
# Generate Quadlet units
quad-ops convert

# The generated units will include:
# - myproject-web-init-0.container (busybox init)
# - myproject-web-init-1.container (alpine init) 
# - myproject-web.container (main nginx service)
# - myproject-database-init-0.container (postgres init)
# - myproject-database.container (main postgres service)

# Start the services
systemctl --user start myproject-web.service
systemctl --user start myproject-database.service
```

## Benefits

- **Simplified Setup**: No need to manage complex shell scripts in main containers
- **Separation of Concerns**: Initialization logic is separate from the main application
- **Dependency Management**: Automatic systemd dependency handling ensures proper startup order
- **Error Handling**: If any init container fails, the main service won't start
