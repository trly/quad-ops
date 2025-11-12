# Quad-Ops Examples

Example Docker Compose configurations demonstrating quad-ops features.

## Examples

### [cross-project-deps/](cross-project-deps/)
Demonstrates cross-project dependencies using the `x-quad-ops-depends-on` extension field. Shows how one project can declare dependencies on services in another project.

### [init-containers/](init-containers/)
Shows how to use init containers that run once before the main service starts.

### [multi-service/](multi-service/)
Basic multi-service application with dependencies, networks, and volumes.

### [phase1-quadlet-deps/](phase1-quadlet-deps/)
Demonstrates Quadlet automatic dependency handling for networks and volumes.

### [device-cgroup-rules/](device-cgroup-rules/)
Shows device cgroup rules configuration for container device access control.

### [namespace-modes/](namespace-modes/)
Demonstrates different namespace modes (PID, IPC, network) for container isolation.

### [sysctls/](sysctls/)
System control parameter configuration examples.

## Running Examples

Each example directory contains a `compose.yml` file and optional README with specific instructions.

### Basic Validation

```bash
cd examples/<example-name>
quad-ops validate compose.yml
```

### Deploy Locally

```bash
cd examples/<example-name>
quad-ops up
```

### Check Status

```bash
# List all services
quad-ops unit list

# Check specific service
systemctl --user status <project>-<service>.service

# View logs
journalctl --user -u <project>-<service>.service -f
```

### Clean Up

```bash
cd examples/<example-name>
quad-ops down
```

## Naming Requirements

All example projects follow quad-ops naming requirements:

- **Project names**: Lowercase letters, digits, dashes, underscores only
- **Service names**: Alphanumeric, dashes, underscores, periods only

See [naming requirements documentation](https://trly.github.io/quad-ops/docs/container-management/naming-requirements/) for details.
