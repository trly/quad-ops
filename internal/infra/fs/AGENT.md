# Agent Guidelines for fs Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `fs` package provides file system operations for quadlet unit file management. It handles unit file paths, content change detection, and safe file writing operations with proper directory creation and permissions.

## Key Functions
- **`GetUnitFilePath(name, unitType string)`** - Returns full path for a quadlet unit file
- **`GetUnitFilesDirectory()`** - Returns directory where quadlet unit files are stored
- **`HasUnitChanged(unitPath, content string)`** - Checks if unit file content has changed
- **`WriteUnitFile(unitPath, content string)`** - Writes unit content to specified file path
- **`GetContentHash(content string)`** - Calculates SHA1 hash for content tracking

## Usage Patterns

### Path Management
```go
// Get unit file path
unitPath := fs.GetUnitFilePath("my-service", "container")
// Returns: /etc/containers/systemd/my-service.container

// Get unit directory
unitDir := fs.GetUnitFilesDirectory()
// Returns: /etc/containers/systemd (or user equivalent)
```

### Change Detection
```go
// Check if unit content has changed
if fs.HasUnitChanged(unitPath, newContent) {
    log.GetLogger().Info("Unit content changed, updating file")
    err := fs.WriteUnitFile(unitPath, newContent)
    if err != nil {
        return fmt.Errorf("failed to write unit file: %w", err)
    }
}
```

## Development Guidelines

### File Path Construction
- Always uses `filepath.Join` for cross-platform compatibility
- Paths are constructed from configuration values, not user input
- Unit file names follow pattern: `{name}.{type}`

### Directory Management
- Automatically creates parent directories with `os.MkdirAll`
- Uses secure permissions: `0750` for directories, `0600` for files
- Handles permission errors gracefully

### Change Detection Strategy
- Uses **direct content comparison** (byte-for-byte) for accuracy
- Reads existing file content and compares with new content
- Missing files are considered "changed" (need to be written)
- SHA1 hashing is used only for logging/debugging, not for change detection

### Security Considerations
- File paths are internally constructed, not user-controlled
- Uses `#nosec G304` annotations where file reading is safe
- Restricts file permissions to prevent unauthorized access
- SHA1 used only for content tracking, not security purposes

## Configuration Integration

### Directory Resolution
- Reads quadlet directory from configuration provider
- Supports both system and user modes
- System mode: `/etc/containers/systemd`
- User mode: `$HOME/.config/containers/systemd`

### Path Construction
```go
func GetUnitFilePath(name, unitType string) string {
    return filepath.Join(
        config.DefaultProvider().GetConfig().QuadletDir,
        fmt.Sprintf("%s.%s", name, unitType),
    )
}
```

## Common Usage Patterns

### Conditional File Updates
```go
// Only write if content has actually changed
if fs.HasUnitChanged(unitPath, newContent) {
    if err := fs.WriteUnitFile(unitPath, newContent); err != nil {
        return fmt.Errorf("failed to update unit file: %w", err)
    }
    log.GetLogger().Info("Updated unit file", "path", unitPath)
} else {
    log.GetLogger().Debug("Unit file unchanged", "path", unitPath)
}
```

### Batch File Operations
```go
// Process multiple units efficiently
for unitName, content := range units {
    unitPath := fs.GetUnitFilePath(unitName, "container")
    if fs.HasUnitChanged(unitPath, content) {
        changedUnits = append(changedUnits, unitName)
        if err := fs.WriteUnitFile(unitPath, content); err != nil {
            errors = append(errors, err)
        }
    }
}
```
