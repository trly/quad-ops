# Agent Guidelines for util Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `util` package provides utility functions for operations like sorting and iterating over slices and maps. It centralizes common patterns used throughout quad-ops to ensure consistency and deterministic output.

## Key Functions
- **`SortAndIterateSlice(slice []string, fn SliceProcessor)`** - Sorts a slice and applies a function to each item
- **`SortStringSlice(slice []string)`** - Sorts a string slice in-place for deterministic order
- **`GetSortedMapKeys(m map[string]string)`** - Returns sorted slice of keys from a map for deterministic iteration
- **`ValidateUnitName(unitName string)`** - Validates systemd unit names for security

### Types
- **`SliceProcessor`** - Function type that processes a string item: `func(string)`

## Usage Patterns

### Deterministic Slice Processing
```go
// Sort and process each item in a slice
sorting.SortAndIterateSlice(labels, func(label string) {
    builder.WriteString(formatKeyValue("Label", label))
})

// Sort slice in-place
sorting.SortStringSlice(slice)
```

### Map Key Sorting
```go
// Get sorted keys for deterministic map iteration
keys := sorting.GetSortedMapKeys(environmentVars)
for _, key := range keys {
    fmt.Printf("%s=%s\n", key, environmentVars[key])
}
```

### Unit Name Validation
```go
// Validate unit name before using in systemd operations
if err := sorting.ValidateUnitName(unitName); err != nil {
    return fmt.Errorf("invalid unit name: %w", err)
}
```

## Development Guidelines

### Deterministic Output Philosophy
The package is designed to ensure consistent, reproducible output across all operations:
- All sorting operations are stable and predictable
- Map iterations use sorted keys to prevent random ordering
- Slice processing maintains original data while providing sorted iteration

### Memory Efficiency
- **Copy-on-Sort**: `SortAndIterateSlice` creates copies to avoid modifying original data
- **In-Place Sorting**: `SortStringSlice` modifies the original slice for efficiency
- **Minimal Allocations**: Functions are designed to minimize memory overhead

### Function Design Patterns
- **Functional Style**: Uses callback functions for flexible processing
- **Safe Defaults**: Handles edge cases like empty slices gracefully
- **Clear Interfaces**: Simple, single-purpose functions

### Unit Name Validation
- Prevents command injection in systemd operations
- Uses regex patterns to ensure safe unit names
- Validates against systemd naming conventions
- Rejects potentially dangerous characters

## Common Patterns

### Deterministic Configuration Generation
```go
// Ensure consistent output order for configuration sections
sorting.SortAndIterateSlice(container.Labels, func(label string) {
    builder.WriteString(formatKeyValue("Label", label))
})

// Process environment variables in sorted order
envKeys := sorting.GetSortedMapKeys(container.Environment)
for _, key := range envKeys {
    fmt.Fprintf(builder, "Environment=%s=%s\n", key, container.Environment[key])
}
```

### Safe Slice Modification
```go
// When you need to sort without modifying original
sorting.SortAndIterateSlice(originalSlice, func(item string) {
    // Process each item in sorted order
    processItem(item)
})

// When in-place sorting is desired
sorting.SortStringSlice(slice) // Modifies original slice
```

### Batch Processing with Consistent Order
```go
// Process multiple collections with deterministic ordering
collections := [][]string{labels, ports, volumes}
for _, collection := range collections {
    sorting.SortAndIterateSlice(collection, func(item string) {
        processConfigItem(item)
    })
}
```

## Performance Considerations

### Sorting Algorithm
- Uses Go's standard `sort.Strings()` which is typically Quicksort/Heapsort hybrid
- Stable sorting ensures consistent results for equal elements
- Efficient for most common use cases in quad-ops

### Memory Usage
- `SortAndIterateSlice` creates temporary copies to preserve original data
- `GetSortedMapKeys` allocates new slice for keys
- Consider using in-place operations when original order doesn't matter

### Scalability
- All functions scale well with typical quad-ops data sizes
- Map key extraction is O(n) where n is map size
- Sorting operations are O(n log n) as expected

## Integration Patterns

### With Configuration Generation
```go
// Use in unit file generation for consistent output
func (u *QuadletUnit) addEnvironmentConfig(builder *strings.Builder) {
    envKeys := sorting.GetSortedMapKeys(u.Container.Environment)
    for _, k := range envKeys {
        fmt.Fprintf(builder, "Environment=%s=%s\n", k, u.Container.Environment[k])
    }
}
```

### With Validation Workflows
```go
// Validate before processing to ensure security
func processUnits(unitNames []string) error {
    for _, name := range unitNames {
        if err := sorting.ValidateUnitName(name); err != nil {
            return fmt.Errorf("invalid unit name %s: %w", name, err)
        }
    }
    
    // Process with sorted order
    sorting.SortAndIterateSlice(unitNames, processUnit)
    return nil
}
```

## Best Practices

### When to Use Each Function
- **`SortAndIterateSlice`**: When you need to process items in order but preserve original
- **`SortStringSlice`**: When you want to permanently sort a slice
- **`GetSortedMapKeys`**: When you need deterministic map iteration

### Error Handling
- Most functions are designed to handle edge cases gracefully
- Validation functions return meaningful errors
- Check for nil inputs where appropriate

### Code Organization
- Import util package for common operations
- Use consistent patterns across similar operations
- Group related utility calls for readability
