# Agent Guidelines for repository Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `repository` package provides data access layer for quad-ops units. It implements a repository pattern for managing unit information with systemd integration, offering filesystem-based scanning and unit metadata management.

## Key Structures and Interfaces

### Core Interfaces
- **`Repository`** - Main interface defining data access operations:
  - `FindAll() ([]Unit, error)`
  - `FindByUnitType(unitType string) ([]Unit, error)`
  - `FindByID(id int64) (Unit, error)`
  - `Create(unit *Unit) (int64, error)`
  - `Delete(id int64) error`

### Core Structures
- **`Unit`** - Represents a managed unit with metadata:
  - `ID` - Unique identifier (hash-based)
  - `Name` - Unit name
  - `Type` - Unit type (container, volume, network, build)
  - `SHA1Hash` - Content hash for change detection
  - `UpdatedAt` - Last modification timestamp

- **`SystemdRepository`** - Implementation using systemd and filesystem scanning:
  - `conn` - systemd dbus connection (optional)

### Key Dependencies
- **`github.com/coreos/go-systemd/v22/dbus`** - systemd integration
- **`internal/fs`** - File system operations
- **`internal/log`** - Centralized logging

## Usage Patterns

### Repository Creation and Usage
```go
// Create repository instance
repo := repository.NewRepository()

// Find all managed units
units, err := repo.FindAll()
if err != nil {
    return fmt.Errorf("failed to find units: %w", err)
}

// Find units by type
containers, err := repo.FindByUnitType("container")
if err != nil {
    return fmt.Errorf("failed to find containers: %w", err)
}
```

### Unit Metadata Access
```go
// Access unit information
for _, unit := range units {
    log.Printf("Unit: %s.%s (ID: %d, Hash: %x, Updated: %v)",
        unit.Name, unit.Type, unit.ID, unit.SHA1Hash, unit.UpdatedAt)
}
```

## Development Guidelines

### Repository Pattern Implementation
- **Interface-based**: Uses Repository interface for testability
- **Filesystem-based**: Scans actual unit files rather than maintaining database
- **systemd Integration**: Leverages systemd for runtime state when needed
- **Content-aware**: Tracks content changes via SHA1 hashing

### Unit Discovery Strategy
- Scans quadlet directory using `filepath.Walk`
- Filters by unit type extensions (.container, .volume, .network, .build)
- Extracts unit names from filenames
- Reads and parses unit files for metadata

### ID Generation
- Uses deterministic hashing based on name+type combination
- Provides consistent IDs across application restarts
- Simple hash function for memory efficiency
- **Note**: Basic hash function may have collision risk for large numbers of units

### Error Handling
- Graceful handling of missing or corrupted unit files
- Continues scanning on individual file errors
- Debug-level logging for non-critical errors
- Clear error messages for critical failures

## Data Access Patterns

### Filesystem-Based Repository
```go
func (r *SystemdRepository) FindByUnitType(unitType string) ([]Unit, error) {
    var units []Unit
    unitFilesDir := fs.GetUnitFilesDirectory()
    
    err := filepath.Walk(unitFilesDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Continue on errors
        }
        
        if !strings.HasSuffix(path, "."+unitType) {
            return nil
        }
        
        unit, err := r.parseUnitFromFile(path, unitName, unitType)
        if err != nil {
            log.GetLogger().Debug("Error parsing unit file", "path", path, "error", err)
            return nil
        }
        
        units = append(units, unit)
        return nil
    })
    
    return units, err
}
```

## Performance Considerations

### Scanning Efficiency
- Uses `filepath.Walk` for efficient directory traversal
- Filters files early to avoid unnecessary processing
- Minimal memory allocation for unit metadata
- Graceful handling of large unit directories

### Memory Usage
- Lightweight Unit structs with minimal overhead
- No caching of file contents (only metadata)
- Efficient string operations for name extraction
- Simple hash function for ID generation

### Connection Management
- Optional systemd connection usage
- Automatic connection cleanup
- No persistent connections maintained
- Lazy connection establishment when needed

## Repository Interface Benefits

### Testability
- Easy mocking for unit tests
- Interface-based dependency injection
- Isolated testing of business logic
- Consistent API across implementations

### Flexibility
- Can be extended with database backends
- Supports multiple implementation strategies
- Easy to add caching layers
- Pluggable architecture for different environments

## Common Usage Patterns

### Unit Enumeration
```go
// Get all units by type for processing
unitTypes := []string{"container", "volume", "network", "build"}
for _, unitType := range unitTypes {
    units, err := repo.FindByUnitType(unitType)
    if err != nil {
        log.GetLogger().Debug("Error finding units by type", "type", unitType, "error", err)
        continue
    }
    // Process units of this type
}
```

### Change Detection
```go
// Compare current content with stored hash
currentHash := fs.GetContentHash(newContent)
if !bytes.Equal(unit.SHA1Hash, currentHash) {
    log.GetLogger().Info("Unit content has changed", "unit", unit.Name)
    // Handle unit update
}
```

### Metadata Tracking
```go
// Access unit metadata for monitoring
for _, unit := range units {
    log.GetLogger().Debug("Unit metadata",
        "name", unit.Name,
        "type", unit.Type,
        "lastModified", unit.UpdatedAt,
        "contentHash", fmt.Sprintf("%x", unit.SHA1Hash))
}
```
