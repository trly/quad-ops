# quad-ops Agent Guidelines

## Commands

- **Build**: `task build` (includes fmt, lint, test, build)
- **Test all**: `task test` or `go test -v ./...`
- **Test single**: `go test -v -run TestFunctionName ./internal/package/`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

## Architecture

GitOps framework converting Docker Compose to systemd Quadlet units with modern dependency injection patterns:

### Core Packages

- `cmd/` - **CLI commands with dependency injection** - All commands use modern DI patterns with no global state
- `internal/compose/` - Docker Compose parsing and conversion to Quadlet units
- `internal/unit/` - Quadlet unit generation (container, network, volume, build)
- `internal/systemd/` - systemd orchestration and service management
- `internal/git/` - Git repository synchronization with configurable backends
- `internal/config/` - Configuration management via Viper with validation
- `internal/fs/` - File system operations with hash-based change detection
- `internal/execx/` - Command execution abstraction for testability
- `internal/testutil/` - Test utilities and helpers for reducing boilerplate
- `internal/validate/` - System validation with dependency injection
- `internal/log/` - Structured logging with configurable levels
- `internal/repository/` - Unit repository abstraction for storage
- `internal/dependency/` - Service dependency graph analysis

### Modern CLI Architecture (All Commands Migrated)

**Dependency Injection Pattern**: All CLI commands use dependency injection

```go
// Modern command structure
type CommandOptions struct {    // Flags bound to struct
    Flag1 string
    Flag2 bool
}

type CommandDeps struct {       // Dependencies injected
    CommonDeps                  // Shared: Clock, FileSystem, Logger
    SpecificDep SomeInterface   // Command-specific dependencies
}

func (c *Command) Run(ctx context.Context, app *App, opts CommandOptions, deps CommandDeps) error {
    // Fully testable implementation with injected dependencies
    return nil
}
```

**Benefits**:

- ✅ **No global state** - Commands are stateless and thread-safe
- ✅ **Fast, reliable tests** - No hanging, no test interference (506+ tests in ~400ms)
- ✅ **Deterministic** - Using `benbjohnson/clock` for time-based testing
- ✅ **Error-based flow** - Commands return errors, main handles exit codes
- ✅ **Mockable dependencies** - All external calls are injected and testable

### Command Architecture Details

**All CLI commands follow this pattern:**

1. **Options Struct** - Command flags bound to structured data (no global variables)
2. **Dependencies Struct** - External dependencies injected for testability
3. **PreRunE/RunE** - Cobra handlers that return errors instead of calling os.Exit
4. **Run Method** - Pure function taking `(ctx, app, options, dependencies)`
5. **buildDeps Method** - Factory for production dependencies

**Migrated Commands:**

- `daemon` - Long-running daemon with periodic sync and systemd integration
- `sync` - Repository synchronization with git and compose processing  
- `up/down` - Service lifecycle management with systemd orchestration
- `doctor` - System validation and health checks
- `unit_*` - Unit inspection and management commands
- `image_pull, root` - Utility commands with proper error handling

## Code Style

### General Guidelines

- Use testify (`assert`/`require`) for tests with descriptive names
- Package comments follow "Package X provides Y" format
- Interface-based design with default providers pattern
- Error handling with early returns, wrap with context
- Use structured logging via `slog`
- Test helpers: `testutil.NewTestLogger()`, `testutil.NewMockConfig()`, temp dirs with cleanup
- Import grouping: stdlib, external, internal
- Constructor injection: Accept dependencies as parameters, no global state
- Command execution: Use `execx.Runner` interface for testable system commands

### Comment Formatting

- ✅ `// performSync executes a sync operation.`
- ❌ `// performSync executes a sync operation`
- All function/type comments must end with periods (godot lint check)

### Parameter Handling

- ✅ `func checkDirectory(_, path string) error`
- ❌ `func checkDirectory(name, path string) error` (when name is unused)
- Use `_` for unused parameters instead of removing them (revive lint check)

### Error Handling

- ✅ `_ = os.Remove(file) // Cleanup - ignore error`
- ❌ `os.Remove(file)` (unchecked)
- Always handle or explicitly ignore error returns (errcheck lint check)
- Document why errors are ignored with comments

### File Operations

- ✅ `os.WriteFile(path, data, 0600)` for sensitive/temporary files
- ❌ `os.WriteFile(path, data, 0644)`
- Use restrictive permissions (0600) for temporary/test files (gosec lint check)

### Memory Allocation

- ✅ `results := make([]CheckResult, 0, expectedSize)`
- ❌ `var results []CheckResult` (when size is known)
- Pre-allocate slices when size is known (prealloc lint check)

### Variable Usage

- ✅ Remove unnecessary assignments or use the modified value
- ❌ Variables assigned but never used in their modified form (ineffassign lint check)

## Common Build Issues and Solutions

### Configuration Access with Viper

**Getting config file path:**

- ❌ `app.ConfigProvider.GetConfig().ConfigFile` (field doesn't exist)
- ✅ `viper.GetViper().ConfigFileUsed()` (gets the actual loaded config file)
- Always check existing API before assuming field names

**Required imports:**

```go
import (
    "github.com/spf13/viper"  // For viper.GetViper()
    "os"                      // For os.Exit(), os.Stat(), etc.
)
```

### Common Import Issues

**Missing standard library imports:**

- Using `os.Exit(1)` requires `"os"` import
- Using `fmt.Printf()` requires `"fmt"` import  
- Using `filepath.Join()` requires `"path/filepath"` import

**Missing external package imports:**

- Using viper functions requires `"github.com/spf13/viper"`
- Using cobra requires `"github.com/spf13/cobra"`

### Type and API Issues

**Undefined methods/fields:**

- Always check existing struct definitions before accessing fields
- Use `Grep` or `Read` tools to verify API signatures
- Look for similar existing code patterns in the codebase

**Build verification:**

- Always run `task build` after code generation
- Fix all lint issues before proceeding
- Check that tests still pass with changes
