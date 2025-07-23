# Agent Guidelines for compose Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `compose` package orchestrates Docker Compose file parsing and conversion to Quadlet units. It handles recursive file discovery, environment variable processing, and project name generation from directory structures.

## Key Functions
- **`ReadProjects(path string)`** - Recursively discovers and reads all Docker Compose projects
- **`ParseComposeFile(path string)`** - Parses single compose file with environment variable substitution
- **`LabelConverter(labels types.Labels)`** - Converts Compose labels to unit labels with deterministic sorting
- **`OptionsConverter(opts map[string]string)`** - Converts driver options with consistent ordering
- **`NameResolver(definedName, keyName string)`** - Resolves resource names from Compose configurations

## Usage Patterns

### Basic Project Reading
```go
projects, err := compose.ReadProjects("/path/to/compose/files")
if err != nil {
    return fmt.Errorf("failed to read projects: %w", err)
}
```

### Environment Variable Processing
- Automatically discovers and loads `.env` files from compose directories
- Validates environment variable keys using POSIX conventions
- Sanitizes sensitive values for logging using `validate.SanitizeForLogging`
- Prevents overriding critical system variables (PATH, HOME, etc.)

## Key Behaviors

### File Discovery
- Supports: `docker-compose.yml`, `docker-compose.yaml`, `compose.yml`, `compose.yaml`
- Uses `filepath.Walk` for recursive scanning
- Continues processing on individual file errors (non-blocking)
- Empty directories return empty slice (not error)

### Project Naming
- Pattern: `repositories/<reponame>/<folder>` → `reponame-folder`
- Falls back to directory name for simple structures
- Names are used for systemd service organization

### Error Handling
- Parsing errors logged at ERROR level but don't stop batch processing
- Directory access errors logged at DEBUG level
- Individual file failures don't prevent other files from processing

## Security Features
- File paths constructed using `filepath.Join` (prevents path traversal)
- Environment variable validation prevents injection attacks
- Secret values sanitized in logs
- File reading uses `#nosec G304` where paths are validated

## Common Pitfalls
- Don't assume all compose files in directory are valid - process continues on errors
- Project names are auto-generated, not user-defined
- Environment variable processing is security-validated, invalid vars are skipped

## Validation Rules
- Environment keys: POSIX naming (alphanumeric + underscore, no digit start)
- Values: 32KB maximum size
- Critical system variables cannot be overridden
- Sensitive values undergo entropy checking and sanitization
