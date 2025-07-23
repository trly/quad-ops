# Agent Guidelines for config Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `config` package provides configuration management for quad-ops using Viper. It handles application settings, repository configurations, and provides a clean interface for accessing configuration throughout the application.

## Key Structures

### Core Interfaces
- **`Provider`** - Main interface for configuration providers with methods:
  - `GetConfig() *Settings`
  - `SetConfig(c *Settings)`
  - `InitConfig() *Settings`
  - `SetConfigFilePath(p string)`

### Configuration Structures
- **`Settings`** - Main configuration structure containing:
  - `RepositoryDir` - Directory for storing repositories
  - `SyncInterval` - How often to sync repositories
  - `QuadletDir` - Directory for systemd unit files
  - `Repositories` - List of managed repositories
  - `UserMode` - Whether running in user mode
  - `Verbose` - Logging verbosity
  - `UnitStartTimeout` - Timeout for unit starts
  - `ImagePullTimeout` - Timeout for container image pulls

- **`Repository`** - Repository configuration containing:
  - `Name` - Repository identifier
  - `URL` - Git repository URL
  - `Reference` - Git reference (branch/tag/commit)
  - `ComposeDir` - Directory within repo containing compose files

## Usage Patterns

### Default Provider Access
```go
// Get current configuration
cfg := config.DefaultProvider().GetConfig()

// Initialize configuration from files
cfg := config.DefaultProvider().InitConfig()

// Set custom config file path
config.DefaultProvider().SetConfigFilePath("/custom/path/config.yaml")
```

## Configuration File Locations
The package searches for configuration files in these locations (in order):
1. `$HOME/.config/quad-ops/config.yaml`
2. `/etc/quad-ops/config.yaml`
3. `./config.yaml` (current directory)

## Default Values
All configuration options have sensible defaults defined as constants:
- `DefaultRepositoryDir` - `/var/lib/quad-ops`
- `DefaultUserRepositoryDir` - `$HOME/.local/share/quad-ops`
- `DefaultQuadletDir` - `/etc/containers/systemd`
- `DefaultUserQuadletDir` - `$HOME/.config/containers/systemd`
- `DefaultSyncInterval` - 5 minutes
- `DefaultUnitStartTimeout` - 10 seconds
- `DefaultImagePullTimeout` - 30 seconds

## Provider Pattern
The package implements a provider pattern allowing for:
- Easy testing with mock providers
- Configuration injection in tests
- Clean separation of concerns

## Error Handling
- Configuration file not found is not an error (uses defaults)
- Invalid configuration files cause panic (fail-fast for misconfigurations)
- Viper unmarshaling errors cause panic

## User Mode vs System Mode
- System mode: Uses `/var/lib/quad-ops` and `/etc/containers/systemd`
- User mode: Uses `$HOME/.local/share/quad-ops` and `$HOME/.config/containers/systemd`
- Automatically adjusts paths based on `userMode` setting

## Repository Configuration Example
```yaml
repositories:
  - name: "my-app"
    url: "https://github.com/user/repo.git"
    ref: "main"
    composeDir: "docker"
  - name: "database"
    url: "git@github.com:org/db-config.git"
    composeDir: "compose/prod"
```

## Common Anti-Patterns to Avoid
- Don't access viper directly throughout the codebase - use the provider interface
- Ensure all new configuration fields have appropriate defaults
- Avoid modifying configuration after initialization - create new instances instead
