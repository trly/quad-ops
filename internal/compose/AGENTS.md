# internal/compose Package

Handles loading, parsing, validating, and processing Docker Compose files into Go data structures compatible with podman-systemd (quadlets).

## Package Responsibilities

The compose package manages the complete lifecycle of compose file processing:

1. **File Discovery** - Locate compose files by name pattern in single or recursive directory walks
2. **File Loading** - Parse YAML and merge environment variables with variable interpolation
3. **Schema Validation** - Validate against the Docker Compose specification
4. **Quadlet Compatibility Validation** - Ensure the project can be converted to podman-systemd units
5. **Dependency Parsing** - Extract intra-project service dependencies

## Core API

### Load(ctx, path, opts) - Single Project

Loads a single compose project from a file or directory path:

- If `path` is a **file**: loads that specific file
- If `path` is a **directory**: searches for `compose.yaml`, `compose.yml`, `docker-compose.yaml`, or `docker-compose.yml` in root only (non-recursive)

**Process**:
1. Determine if path is file or directory
2. Locate compose file using standard naming conventions
3. Establish working directory for relative path resolution
4. Load and merge environment variables from `.env` files and options
5. Parse YAML using compose-go loader (with validation deferred)
6. Set project name from directory if not explicitly defined
7. Run full compose spec validation
8. Validate quadlet compatibility constraints
9. Parse service dependencies (internal and external)

Returns `*types.Project` on success or typed error on failure.

### LoadAll(ctx, path, opts) - Recursive Discovery

Recursively discovers and loads all compose projects in a directory tree:

- Walks the entire directory tree looking for compose files
- Continues on individual project load errors (collects errors)
- Returns `[]LoadedProject` with both successful and failed projects

**Process**:
1. Verify path exists and is a directory
2. Recursively find all compose files matching naming conventions
3. Load each compose file individually using `Load()`
4. Collect results and errors
5. Return all projects (both successful and failed)

## Configuration

### LoadOptions

Controls compose file loading behavior:

```go
type LoadOptions struct {
    Workdir     string            // Base directory for relative path resolution (default: file's directory)
    Environment map[string]string // Environment variables for interpolation (overrides .env files)
    EnvFiles    []string          // Additional .env files to load
}
```

**Environment Resolution Order** (lowest to highest precedence):
1. Environment variables from specified `EnvFiles`
2. Environment variables from default `.env` in workdir
3. Environment variables from `Environment` option map

## Validation Pipeline

### 1. Compose Specification Validation

Uses compose-go's schema validation to ensure:
- Valid YAML syntax
- Required fields present
- Field types correct
- Values within expected ranges

### 2. Quadlet Compatibility Validation

Ensures the project can be converted to podman-systemd units. Validates:

**Service-Level Checks**:
- Must have explicit image or Dockerfile (no image auto-detection)
- cap_add and cap_drop are supported (mapped to AddCapability/DropCapability)
- security_opt: only specific values supported (mapped to Quadlet keys):
  - label=disable, label=nested, label=type:*, label=level:*, label=filetype:* → SecurityLabel* keys
  - no-new-privileges → NoNewPrivileges
  - seccomp=* → SeccompProfile
  - mask=*, unmask=* → Mask/Unmask
  - Other security_opt values (apparmor, etc.) are rejected
- No privileged mode or user specification (use systemd user mapping)
- No container-specific IPC sharing (only private/shareable)
- Supported restart policies: no, always, on-failure, unless-stopped
- No deployment replicas > 1
- No placement constraints/preferences
- Network mode must be bridge or host (not none or container-specific)
- Cannot publish ports in host network mode
- Only service_started supported for depends_on conditions
- Logging drivers: json-file or journald only
- Stop signals: SIGTERM or SIGKILL only
- No tmpfs mounts
- No service profiles

**Project-Level Checks**:
- Volume drivers: local only
- Network drivers: bridge only

### 3. Dependency Parsing

Extracts and validates service dependencies:

**Intra-Project Dependencies** (from `depends_on`):
- Map of service name to condition (defaults to "service_started")

## Error Types

All errors implement standard Go error interface and support `errors.As()` for type checking:

- **fileNotFoundError** - Compose file not found at specified path
- **pathError** - Directory/file access error (permissions, invalid path)
- **invalidYAMLError** - YAML parsing error
- **loaderError** - Compose file parsing/loading failure
- **validationError** - Compose spec validation failure
- **quadletCompatibilityError** - Project not compatible with quadlet conversion

Each error type provides a public checker function: `IsFileNotFoundError()`, `IsValidationError()`, etc.

## File Discovery

### Single File Search (Load)

For a directory path, searches in order:
1. `compose.yaml`
2. `compose.yml`
3. `docker-compose.yaml`
4. `docker-compose.yml`

Stops at first match. Returns error if none found.

### Recursive Search (LoadAll)

Walks entire directory tree and collects all matching compose files. Non-blocking walk (continues on subdirectory errors).

## Environment Variable Handling

### .env File Format

Standard shell-like format supported:
- `KEY=value`
- `KEY="value with spaces"`
- Comments with `#`
- Blank lines

### Variable Interpolation

Variables referenced in compose files as `${VAR}` or `$VAR` are resolved:
- Using compose-go's built-in interpolation
- Against the merged environment map
- Undefined variables remain unchanged in the file

### Context Awareness

All public functions are context-aware:
- Accept `context.Context` as first parameter
- Check context cancellation before work
- Return `ctx.Err()` if context is cancelled

## Testing

The package includes `load_test.go` with comprehensive test coverage for:
- Single file and directory loading
- Recursive directory scanning
- Environment variable handling
- Validation errors
- Error type detection

Tests validate the public API behavior and error contract without exposing internal implementation details.

To verify changes to compose validation and loading, see the [systemd package AGENTS.md](../systemd/AGENTS.md) section on "Validation Using podman-systemd-generator" for how to test the complete compose→quadlet conversion pipeline.
