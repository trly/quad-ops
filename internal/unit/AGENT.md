# Agent Guidelines for unit Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `unit` package provides Quadlet unit definitions and generation functionality for quad-ops. It handles the conversion of Docker Compose configurations to systemd Quadlet unit files, supporting containers, volumes, networks, and build units.

## Key Structures and Interfaces

### Core Structures
- **`QuadletUnit`** - Main unit structure containing:
  - `Name` - Unit name
  - `Type` - Unit type (container, volume, network, build)
  - `Systemd` - systemd configuration
  - `Container/Volume/Network/Build` - Type-specific configuration

- **`SystemdConfig`** - systemd unit configuration:
  - `Description` - Unit description
  - `After/Before` - Dependency ordering
  - `Requires/Wants` - Dependency relationships
  - `RestartPolicy` - Restart behavior
  - `TimeoutStartSec` - Start timeout

### Unit Types
- **`Container`** - Container unit with Docker Compose conversion support
- **`Volume`** - Volume unit for persistent storage
- **`Network`** - Network unit for container networking
- **`Build`** - Build unit for container image building

### Key Dependencies
- **`github.com/compose-spec/compose-go/v2/types`** - Docker Compose specification
- **`internal/systemd`** - systemd unit interface implementation
- **`internal/sorting`** - Sorting and utility functions
- **`internal/validate`** - Security validation
- **`internal/log`** - Centralized logging

## Usage Patterns

### Unit Creation and Generation
```go
// Create a container unit
container := unit.NewContainer("my-service")
container.FromComposeService(composeService, project)

// Generate Quadlet unit file content
content := unit.GenerateQuadletUnit(QuadletUnit{
    Name: "my-service",
    Type: "container",
    Container: *container,
})
```

### Docker Compose Conversion
```go
// Convert a Compose service to Container unit
container := unit.NewContainer(serviceName)
container.FromComposeService(service, project)

// Handle init containers
initContainers, err := unit.ParseInitContainers(service)
if err != nil {
    return fmt.Errorf("failed to parse init containers: %w", err)
}
```

## Development Guidelines

### Docker Compose Conversion
The package provides comprehensive conversion from Docker Compose to Quadlet format:
- **Environment Variables**: Sorted deterministically with security validation
- **Volumes**: Handles bind mounts, named volumes, and external volumes
- **Networks**: Supports project networks, external networks, and aliases
- **Health Checks**: Converts Docker health checks to Quadlet format
- **Resource Constraints**: Maps to PodmanArgs for unsupported features
- **Secrets**: Supports both file-based and environment variable secrets

### Deterministic Output
All generated unit files have deterministic content through:
- **Sorted Maps**: Environment variables, sysctls, log options
- **Sorted Slices**: Labels, ports, volumes, networks
- **Consistent Ordering**: All configuration sections in predictable order
- **Key Normalization**: Consistent key formatting and casing

## Container Unit Features

### Comprehensive Compose Support
- **Basic Configuration**: Image, labels, commands, environment
- **Networking**: Port publishing, network connections, aliases
- **Storage**: Volume mounts, tmpfs, bind mounts
- **Health Checks**: Command-based health monitoring
- **Resource Limits**: Memory, CPU, PID limits (via PodmanArgs)
- **Security**: Capabilities, security contexts, secrets
- **Advanced Options**: Ulimits, sysctls, logging configuration

### Init Container Support
- **Extension-Based**: Uses `x-quad-ops-init` extension
- **Inheritance**: Init containers inherit parent configuration
- **Dependency Management**: Proper systemd dependencies
- **Resource Sharing**: Volumes, networks, secrets shared with parent

### PodmanArgs Integration
For Docker Compose features not directly supported by Quadlet:
- **Automatic Fallback**: Unsupported features mapped to PodmanArgs
- **Warning Logging**: Clear indication of fallback usage
- **Feature Preservation**: All functionality maintained through Podman CLI
- **Security Maintained**: Validation still applied to fallback options

## Volume and Network Units

### Volume Unit Features
- **Named Volumes**: Project-scoped volume management
- **External Volumes**: Reference to pre-existing volumes
- **Driver Options**: Support for volume driver configuration
- **Labels**: Metadata and management labels
- **Copy Behavior**: Control initial content copying

### Network Unit Features
- **Custom Networks**: Project-specific network creation
- **External Networks**: Reference to pre-existing networks
- **IPAM Configuration**: IP address management settings
- **Driver Options**: Network driver configuration
- **Isolation**: Internal network support

## Common Patterns

### Docker Compose Service Conversion
```go
func (c *Container) FromComposeService(service types.ServiceConfig, project *types.Project) *Container {
    // Initialize required fields
    c.RunInit = new(bool)
    *c.RunInit = true

    // Process configuration sections
    c.setBasicServiceFields(service)
    c.processServicePorts(service)
    c.processServiceEnvironment(service)
    c.processServiceVolumes(service, project)
    c.processServiceNetworks(service, project)
    c.processServiceHealthCheck(service)
    c.processServiceResources(service)
    c.processAdvancedConfig(service)
    c.processServiceSecrets(service)

    // Ensure deterministic output
    sortContainer(c)
    return c
}
```

### Unit File Generation
```go
func GenerateQuadletUnit(unit QuadletUnit) string {
    content := unit.generateUnitSection()

    switch unit.Type {
    case "container":
        content += unit.generateContainerSection()
    case "volume":
        content += unit.generateVolumeSection()
    case "network":
        content += unit.generateNetworkSection()
    case "build":
        content += unit.generateBuildSection()
    }

    content += unit.generateServiceSection()
    return content
}
```

## Build Unit Support

### Build Configuration
- **Context and Dockerfile**: Build context and Dockerfile specification
- **Image Tags**: Multiple tag support for built images
- **Build Arguments**: Environment variables for build process
- **Resource Constraints**: Build-time resource limits
- **Network Access**: Network configuration during build
- **Volume Mounts**: Build-time volume access
- **Secret Access**: Build-time secret availability

### Build Metadata
- **Labels**: Image metadata and annotations
- **Annotations**: Additional image metadata
- **Target Stage**: Multi-stage build target selection
- **Pull Policy**: Base image pull behavior

## Performance Considerations

### Memory Efficiency
- **Lazy Initialization**: Complex structures created on demand
- **Efficient Sorting**: In-place sorting where possible
- **Minimal Allocations**: Reuse of slices and maps
- **Deterministic Maps**: Pre-sorted key slices for map iteration

### Generation Speed
- **Template-Free**: Direct string building for speed
- **Batch Operations**: Process multiple units efficiently
- **Minimal I/O**: Generate content in memory before writing
- **Parallel Processing**: Unit generation can be parallelized

## Error Handling

### Validation Errors
- **Non-Fatal**: Invalid configurations logged but don't stop processing
- **Security Failures**: Security violations are logged and skipped
- **Conversion Warnings**: Unsupported features generate warnings
- **Context Preservation**: Error messages include sufficient context

### Recovery Strategies
- **Graceful Degradation**: Skip invalid items, continue processing
- **Default Values**: Secure defaults for missing configuration
- **Fallback Mechanisms**: PodmanArgs for unsupported features
- **User Feedback**: Clear indication of what was skipped and why
