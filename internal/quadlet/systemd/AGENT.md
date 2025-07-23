# Agent Guidelines for systemd Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `systemd` package provides systemd unit management operations and orchestration for quad-ops. It handles unit lifecycle operations, dependency-aware restarts, and systemd integration through D-Bus connections.

## Key Structures and Interfaces

### Core Interfaces
- **`Unit`** - Main interface for systemd unit operations:
  - `GetServiceName() string` - Returns full systemd service name
  - `GetUnitType() string` - Returns unit type (container, volume, network, etc.)
  - `GetUnitName() string` - Returns unit name
  - `GetStatus() (string, error)` - Returns current unit status
  - `Start() error` - Starts the unit
  - `Stop() error` - Stops the unit
  - `Restart() error` - Restarts the unit
  - `Show() error` - Displays unit configuration and status
  - `ResetFailed() error` - Resets failed state

### Core Structures
- **`BaseUnit`** - Common implementation for all systemd units:
  - `Name` - Unit name
  - `Type` - Unit type

- **`OrchestrationResult`** - Result of orchestration operations:
  - `Success` - Whether operation succeeded
  - `Errors` - Map of unit names to errors

- **`UnitChange`** - Represents a unit that has changed:
  - `Name` - Unit name
  - `Type` - Unit type
  - `Unit` - Unit interface instance

### Key Dependencies
- **`github.com/coreos/go-systemd/v22/dbus`** - systemd D-Bus integration
- **`internal/config`** - Configuration access
- **`internal/dependency`** - Service dependency management
- **`internal/log`** - Centralized logging

## Usage Patterns

### Basic Unit Operations
```go
// Create a unit
unit := systemd.NewBaseUnit("my-service", "container")

// Check status
status, err := unit.GetStatus()
if err != nil {
    return fmt.Errorf("failed to get status: %w", err)
}

// Start the unit
err = unit.Start()
if err != nil {
    return fmt.Errorf("failed to start unit: %w", err)
}
```

### Dependency-Aware Operations
```go
// Start with dependency awareness
err := systemd.StartUnitDependencyAware("web-service", "container", dependencyGraph)
if err != nil {
    return fmt.Errorf("failed to start unit: %w", err)
}

// Restart changed units with orchestration
changedUnits := []systemd.UnitChange{
    {Name: "web-service", Type: "container", Unit: webUnit},
    {Name: "db-service", Type: "container", Unit: dbUnit},
}
err := systemd.RestartChangedUnits(changedUnits, projectDependencyGraphs)
```

## Development Guidelines

### Service Name Generation
- Container units: `{name}.service`
- Other units: `{name}-{type}.service`
- Examples: `web.service`, `db-volume.service`, `app-network.service`

### Connection Management
- Uses D-Bus connections to systemd
- Supports both system and user mode connections
- Connections are opened per operation and closed immediately
- User mode: `dbus.NewUserConnectionContext()`
- System mode: `dbus.NewSystemConnectionContext()`

### Error Handling Strategy
- Graceful handling of unit state transitions
- Detailed error reporting with failure context
- Timeout handling for long-running operations
- Special handling for "activating" states

### Orchestration Patterns
- One-shot services (volume, network, build) started first
- Container services restarted with dependency awareness
- Async restart initiation for better performance
- Final status verification for all units

## Common Patterns

### Safe Unit Operations
```go
func (u *BaseUnit) Start() error {
    conn, err := getSystemdConnection()
    if err != nil {
        return fmt.Errorf("error connecting to systemd: %w", err)
    }
    defer conn.Close()

    serviceName := u.GetServiceName()
    log.GetLogger().Debug("Attempting to start unit", "name", serviceName)

    ch := make(chan string)
    _, err = conn.StartUnitContext(context.Background(), serviceName, "replace", ch)
    if err != nil {
        return fmt.Errorf("error starting unit %s: %w", serviceName, err)
    }

    result := <-ch
    if result != "done" {
        // Handle activation states and provide detailed errors
        details := getUnitFailureDetails(serviceName)
        return fmt.Errorf("unit start failed: %s%s", result, details)
    }

    return nil
}
```

### Connection Management
```go
func getSystemdConnection() (*dbus.Conn, error) {
    cfg := config.DefaultProvider().GetConfig()
    
    if cfg.UserMode {
        log.GetLogger().Debug("Establishing user connection to systemd")
        return dbus.NewUserConnectionContext(ctx)
    }
    
    log.GetLogger().Debug("Establishing system connection to systemd")
    return dbus.NewSystemConnectionContext(ctx)
}
```

## Orchestration Features

### Async Restart Handling
- Initiates restarts without blocking on completion
- Monitors unit states for final verification
- Handles timeout scenarios gracefully
- Provides detailed failure diagnostics

### State Transition Management
- Recognizes "activating" states as valid intermediate states
- Different timeouts for different operations (start vs image pull)
- Proper handling of systemd unit lifecycle
- Clear logging for debugging state issues

### Failure Diagnostics
- Retrieves unit properties via D-Bus
- Extracts recent logs using journalctl
- Provides comprehensive error context
- Safe unit name validation for log commands

## Configuration Integration

### Timeout Configuration
- `UnitStartTimeout` - General unit start timeout
- `ImagePullTimeout` - Extended timeout for container image operations
- Configurable via application settings
- Used in activation state waiting logic

### Mode Selection
- System mode: Uses system D-Bus connection
- User mode: Uses user D-Bus connection
- Affects unit file locations and permissions
- Controls journalctl command selection

## Performance Considerations

### Async Operations
- Non-blocking restart initiation
- Batch processing of changed units
- Minimal connection overhead
- Efficient state polling

### Resource Management
- Short-lived D-Bus connections
- Proper connection cleanup
- Minimal memory footprint
- Efficient error propagation
