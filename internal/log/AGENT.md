# Agent Guidelines for log Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `log` package provides centralized logging functionality for quad-ops using Go's structured logging library (`log/slog`). It manages application-wide logging configuration with support for different verbosity levels and consistent output formatting.

## Key Functions
- **`Init(verbose bool)`** - Initializes the global logger with appropriate verbosity level
- **`GetLogger()`** - Returns the configured logger instance for use throughout the application

## Configuration
- **Verbosity Control**: 
  - Normal mode: `slog.LevelInfo` (Info, Warn, Error messages)
  - Verbose mode: `slog.LevelDebug` (All messages including Debug)
- **Output Format**: Text handler for human-readable console output
- **Global Logger**: Sets as default slog logger for the entire application

## Usage Patterns

### Logger Initialization
```go
// Initialize with normal verbosity (Info level and above)
log.Init(false)

// Initialize with verbose output (Debug level and above)
log.Init(true)
```

### Getting Logger Instance
```go
// Get the configured logger
logger := log.GetLogger()
logger.Info("Application started", "version", "1.0.0")
logger.Debug("Debug information", "key", "value")
logger.Error("An error occurred", "error", err)
```

### Structured Logging
```go
// Use key-value pairs for structured data
log.GetLogger().Info("Processing compose file", 
    "path", "/path/to/compose.yml",
    "project", "my-project",
    "services", 3)

log.GetLogger().Error("Failed to sync repository",
    "repo", "my-repo",
    "url", "https://github.com/user/repo.git",
    "error", err)
```

## Development Guidelines

### Initialization Strategy
- Logger must be initialized once at application startup
- Call `log.Init()` before any other package that uses logging
- Verbosity is typically controlled by command-line flags or configuration

### Global Logger Pattern
- Package maintains a single global logger instance
- `slog.SetDefault()` makes it available to other packages that use slog directly
- Provides consistent logging behavior across the entire application

### Message Levels
- **Debug**: Detailed diagnostic information, only visible in verbose mode
- **Info**: General application flow and important events
- **Warn**: Warning conditions that don't stop execution
- **Error**: Error conditions that may affect functionality

### Structured Logging Best Practices
- Always use key-value pairs for contextual information
- Keep keys consistent across similar log messages
- Use meaningful, descriptive keys
- Include relevant context (file paths, service names, error details)

## Configuration Integration

### Verbosity Control
```go
// From configuration or command-line flags
verbose := config.DefaultProvider().GetConfig().Verbose
log.Init(verbose)
```

### Dynamic Log Levels
- Current implementation uses static log levels set at initialization
- For dynamic level changes, would need to recreate the logger
- Consider implementing level change functionality if needed

## Common Usage Patterns

### Standard Application Logging
```go
// Application startup
log.GetLogger().Info("Starting quad-ops", "version", version)

// Processing operations
log.GetLogger().Debug("Processing repository", "name", repoName)

// Error handling
if err != nil {
    log.GetLogger().Error("Operation failed", "operation", "sync", "error", err)
    return err
}

// Success operations
log.GetLogger().Info("Repository synced successfully", "name", repoName, "duration", duration)
```

### Debug Logging
```go
// Only visible when verbose mode is enabled
log.GetLogger().Debug("Parsing compose file", 
    "path", filePath,
    "size", fileSize,
    "modified", modTime)
```

### Error Context
```go
// Include sufficient context for debugging
log.GetLogger().Error("Failed to write unit file",
    "unit", unitName,
    "type", unitType,
    "path", unitPath,
    "error", err)
```

## Performance Considerations

### Minimal Overhead
- slog is designed for high-performance structured logging
- Debug messages are efficiently filtered when not enabled
- Structured logging avoids string formatting unless needed

### Memory Usage
- Logger uses minimal memory for configuration
- Log messages are written directly to stdout
- No internal buffering or complex processing

## Best Practices

### Consistent Key Names
- Use consistent keys across the application: `"error"`, `"path"`, `"name"`, `"type"`
- Establish naming conventions for common concepts
- Document key naming standards for the team

### Appropriate Log Levels
- Debug: Internal state, detailed flow information
- Info: User-visible operations, important state changes
- Warn: Recoverable issues, deprecated functionality
- Error: Failures that affect functionality

### Context Inclusion
- Always include relevant context with error messages
- Provide enough information for debugging without being verbose
- Include operation names, resource identifiers, and timing information
