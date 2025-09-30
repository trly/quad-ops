# CLI Testing Guide for @cmd Package

This guide explains how to structure CLI tests using modern dependency injection patterns for robust, maintainable testing.

## Testing Architecture Overview

The cmd package uses **dependency injection** and **error-based flow control** to test CLI commands without requiring OS dependencies (systemd, podman, git, etc.).

### Core Principles ✅ PROVEN WORKING

1. **Thin Cobra Commands** - Keep commands as adapters that parse flags and call runners
2. **Dependency Injection** - Pass dependencies explicitly, avoid global mutable state
3. **Error Propagation** - Use RunE/PreRunE and return errors instead of os.Exit
4. **Deterministic Testing** - Use fake time (`benbjohnson/clock`) and controllable dependencies
5. **Fast, Non-Hanging Tests** - Use short timeouts and context cancellation
6. **Follow Go Conventions** - Each `*.go` file has corresponding `*_test.go`

### Successfully Migrated Commands

✅ **daemon** - Full dependency injection with 9 test cases, no hanging  
✅ **sync** - Architecture refactored with DI framework  
✅ **up, down** - Service lifecycle management with comprehensive tests
✅ **doctor** - System validation with 17 test cases covering all check scenarios
✅ **version** - Version display with 6 test cases including update checking
✅ **validate** - Compose validation with 13 test cases covering all validation paths
✅ **image_pull** - Image pulling with 12 test cases (2 skipped for integration)
✅ **unit_*, root, config** - All modernized with comprehensive tests

**🎉 ALL COMMANDS SUCCESSFULLY MIGRATED TO MODERN DEPENDENCY INJECTION PATTERN**

**Test Coverage**: 579 tests running in ~400ms with 72.9% cmd package coverage

## Code Guidelines

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

### Directory Permissions
- ✅ `os.MkdirAll(path, 0750)` for test directories
- ❌ `os.MkdirAll(path, 0755)` 
- Use 0750 or less for directories in tests (gosec G301 check)

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
- Using `clock.New()` requires `"github.com/benbjohnson/clock"` import

**Missing external package imports:**
- Using viper functions requires `"github.com/spf13/viper"`
- Using cobra requires `"github.com/spf13/cobra"`

### FileSystem Mocking Issues

**Cannot assign to interface fields directly:**
```go
// ❌ WRONG - Cannot assign to interface method
deps.FileSystem.WriteFile = func(...) error { return nil }

// ✅ CORRECT - Create new FileSystemOps with mock functions
mockFS := &FileSystemOps{
    StatFunc: func(path string) (fs.FileInfo, error) {
        return os.Stat(path)
    },
    WriteFileFunc: func(_ string, _ []byte, _ fs.FileMode) error {
        return errors.New("permission denied")
    },
    RemoveFunc: func(path string) error {
        return os.Remove(path)
    },
    MkdirAllFunc: func(path string, perm fs.FileMode) error {
        return os.MkdirAll(path, perm)
    },
}

deps := CommandDeps{
    CommonDeps: CommonDeps{
        Clock:      clock.New(),
        FileSystem: mockFS,  // Inject the mock
        Logger:     testutil.NewTestLogger(t),
    },
}
```

### exec.Cmd and Context Issues

**Command creation for tests:**
```go
// ❌ WRONG - Setting Cancel field directly causes errors
cmd := exec.Command("echo", "test")
cmd.Cancel = func() error { return cmd.Process.Kill() }

// ✅ CORRECT - Use CommandContext for commands that need cancellation
cmd := exec.CommandContext(ctx, "echo", "test")

// ✅ CORRECT - Or use simple Command for tests without cancellation
cmd := exec.Command("echo", "test")
```

**Mocking exec.Command properly:**
```go
// Capture arguments for verification instead of asserting in mock
var argsReceived []string
deps := CommandDeps{
    ExecCommand: func(name string, args ...string) *exec.Cmd {
        argsReceived = args  // Capture for later verification
        return exec.Command("echo", "success")
    },
}

// Verify after execution
assert.Contains(t, argsReceived, "expected-arg")
```

### Type and API Issues

**Undefined methods/fields:**
- Always check existing struct definitions before accessing fields
- Use `Grep` or `Read` tools to verify API signatures
- Look for similar existing code patterns in the codebase

**Build verification:**
- Always run `task build` after code generation
- Fix all lint issues before proceeding
- Check that tests still pass with changes

### Unused Variable Issues

**Avoid declaring variables that aren't used:**
```go
// ❌ WRONG - Variable declared but not meaningfully used
var envSet bool
deps.ExecCommand = func(...) *exec.Cmd {
    if len(cmd.Env) > 0 {
        envSet = true  // Set but never checked
    }
}

// ✅ CORRECT - Only declare if you verify the value
var envWasSet bool
// ... set the variable ...
assert.True(t, envWasSet)  // Actually verify it

// ✅ CORRECT - Or don't declare it if not needed
deps.ExecCommand = func(...) *exec.Cmd {
    // Just execute without tracking
}
```

## Agent Collaboration Workflow

### Quality Gates

- No reduction in test coverage
- Security review for authentication, authorization, or data handling
- Style and lint checks must pass

## File Organization

```
cmd/
├── up.go              ← Command implementation  
├── up_test.go         ← Tests for up command
├── sync.go            ← Command implementation
├── sync_test.go       ← Tests for sync command
├── mocks_test.go      ← Mock implementations (shared)
├── test_helpers.go    ← Test utilities (shared)
├── interfaces.go      ← Test interfaces (shared)
└── deps.go            ← Shared dependency types (optional)
```

## Command Implementation Guidelines

### Mandatory Pattern for ALL Commands

**Every command in the cmd package MUST follow this modern dependency injection pattern:**

**Benefits of This Pattern:**
- ✅ **No global state** - Commands are stateless and thread-safe
- ✅ **Fast, reliable tests** - No hanging, no test interference (506+ tests in ~400ms)
- ✅ **Deterministic** - Using `benbjohnson/clock` for time-based testing
- ✅ **Error-based flow** - Commands return errors, main handles exit codes
- ✅ **Mockable dependencies** - All external calls are injected and testable

```go
// Command structure (stateless)
type CommandName struct{}

// Options struct (replaces global flags)
type CommandNameOptions struct {
    Flag1 Type1
    Flag2 Type2
}

// Dependencies struct (replaces global seams)
type CommandNameDeps struct {
    CommonDeps              // Always include shared dependencies
    SpecificDep Interface   // Command-specific dependencies
}

// PreRunE/RunE pattern (replaces PreRun/Run + os.Exit)
PreRunE: func(cmd *cobra.Command, _ []string) error {
    app := c.getApp(cmd)
    return app.Validator.SystemRequirements()
},
RunE: func(cmd *cobra.Command, _ []string) error {
    app := c.getApp(cmd)
    deps := c.buildDeps(app)
    return c.Run(cmd.Context(), app, opts, deps)
},

// Testable runner
func (c *CommandName) Run(ctx context.Context, app *App, opts CommandNameOptions, deps CommandNameDeps) error {
    // Implementation - fully testable
    return nil
}
```

### 1. Command Structure

Follow this pattern for all new commands:

```go
// Command struct (no state)
type ExampleCommand struct{}

func NewExampleCommand() *ExampleCommand {
    return &ExampleCommand{}
}

// Command options (bound to flags)
type ExampleOptions struct {
    Verbose    bool
    OutputPath string
    Force      bool
}

// Dependencies (injected, not global)
type ExampleDeps struct {
    Clock      clock.Clock
    FileSystem FileSystemOps
    Logger     log.Logger
    Notify     func(bool, string) (bool, error)
}

// Cobra command factory
func (c *ExampleCommand) GetCobraCommand() *cobra.Command {
    var opts ExampleOptions
    
    cmd := &cobra.Command{
        Use:   "example",
        Short: "Example command description",
        PreRunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            return app.Validator.SystemRequirements()
        },
        RunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            deps := c.buildDeps(app)
            return c.Run(cmd.Context(), app, opts, deps)
        },
        SilenceUsage:  true, // Don't show usage on errors
        SilenceErrors: true, // Let main handle error display
    }
    
    // Bind flags to options struct
    cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
    cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Output file path")
    cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force operation")
    
    return cmd
}

// Testable runner (pure function)
func (c *ExampleCommand) Run(ctx context.Context, app *App, opts ExampleOptions, deps ExampleDeps) error {
    // Implementation here - fully testable
    return nil
}

// Dependency builder (production dependencies)
func (c *ExampleCommand) buildDeps(app *App) ExampleDeps {
    return ExampleDeps{
        Clock:      clock.New(),
        FileSystem: FileSystemOps{
            Stat:      os.Stat,
            WriteFile: os.WriteFile,
            Remove:    os.Remove,
            MkdirAll:  os.MkdirAll,
        },
        Logger: app.Logger,
        Notify: daemon.SdNotify,
    }
}

// App context helper
func (c *ExampleCommand) getApp(cmd *cobra.Command) *App {
    return cmd.Context().Value(appContextKey).(*App)
}
```

### 2. Dependency Types

Define shared dependency types for consistency:

```go
// deps.go (optional shared file)
type FileSystemOps struct {
    Stat      func(string) (fs.FileInfo, error)
    WriteFile func(string, []byte, fs.FileMode) error
    Remove    func(string) error
    MkdirAll  func(string, fs.FileMode) error
}

type ProcessOps struct {
    Start func(*exec.Cmd) error
    Wait  func(*exec.Cmd) error
    Kill  func(*os.Process) error
}

type NetworkOps struct {
    HTTPGet  func(string) (*http.Response, error)
    HTTPPost func(string, io.Reader) (*http.Response, error)
}
```

### 3. Error Handling

Always return errors, never call `os.Exit`:

```go
// ❌ Don't do this
if err := doSomething(); err != nil {
    app.Logger.Error("Failed to do something", "error", err)
    os.Exit(1)
}

// ✅ Do this instead
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### 4. Context Usage

Always use context for cancellation and timeouts:

```go
func (c *ExampleCommand) Run(ctx context.Context, app *App, opts ExampleOptions, deps ExampleDeps) error {
    // Check context early
    if err := ctx.Err(); err != nil {
        return err
    }
    
    // Pass context to long-running operations
    if err := c.performLongOperation(ctx, deps); err != nil {
        return err
    }
    
    return nil
}

func (c *ExampleCommand) performLongOperation(ctx context.Context, deps ExampleDeps) error {
    ticker := deps.Clock.Ticker(time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C():
            // Do work
        }
    }
}
```

## Modern Command Architecture

### 1. Command Structure

Structure commands using dependency injection and error propagation:

```go
// Command options (no global state)
type DaemonOptions struct {
    SyncInterval time.Duration
    RepoName     string
    Force        bool
}

// Injected dependencies
type DaemonDeps struct {
    Clock      clock.Clock                                      // benbjohnson/clock for deterministic time
    Notify     func(bool, string) (bool, error)                // systemd notifications  
    MkdirAll   func(string, os.FileMode) error                 // file operations
    Logger     log.Logger
}

func (c *DaemonCommand) GetCobraCommand() *cobra.Command {
    var opts DaemonOptions
    
    cmd := &cobra.Command{
        Use: "daemon",
        PreRunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            return app.Validator.SystemRequirements() // Return error, don't exit
        },
        RunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            deps := c.buildDeps(app) // Inject dependencies
            return c.Run(cmd.Context(), app, opts, deps)
        },
    }
    
    // Bind flags to options struct
    cmd.Flags().DurationVarP(&opts.SyncInterval, "sync-interval", "i", 5*time.Minute, "Sync interval")
    return cmd
}

// Testable runner with injected dependencies
func (c *DaemonCommand) Run(ctx context.Context, app *App, opts DaemonOptions, deps DaemonDeps) error {
    // Implementation here - fully testable with fake dependencies
}
```

### 2. AppBuilder Pattern

Use the fluent `AppBuilder` to construct test apps with mocked dependencies:

```go
app := NewAppBuilder(t).
    WithValidator(&MockValidator{
        SystemRequirementsFunc: func() error {
            return errors.New("systemd not found") // Simulate failure
        },
    }).
    WithUnitRepo(mockRepo).
    WithUnitManager(mockManager).
    WithConfig(customConfig).
    Build(t)
```

### 3. Error-Based Testing

Commands return errors instead of calling `os.Exit()`:

```go
func TestCommand_ValidationFailure(t *testing.T) {
    app := NewAppBuilder(t).
        WithValidator(&MockValidator{
            SystemRequirementsFunc: func() error {
                return errors.New("systemd not found")
            },
        }).
        Build(t)

    cmd := NewCommand().GetCobraCommand()
    SetupCommandContext(cmd, app)
    
    // PreRunE returns error instead of exiting
    err := cmd.PreRunE(cmd, []string{})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "systemd not found")
}
```

### 4. Output Capture

Use the test helpers for reliable output capture:

```go
// For output capture (handles both fmt.Print* and cmd.Print*)
output, err := ExecuteCommandWithCapture(t, cmd, []string{"arg1", "arg2"})
assert.Contains(t, output, "Expected output text")

// For simple execution without output
err := ExecuteCommand(t, cmd, []string{"arg1"})
assert.NoError(t, err)

// For command setup
SetupCommandContext(cmd, app)
```

## Test Seams for OS Dependencies

⚠️ **DEPRECATED PATTERN** - Use dependency injection instead of global seams.

### Legacy Pattern (Avoid)
```go
// ❌ Global seams - brittle and causes test interference
var exitFunc = os.Exit        // up.go, down.go
var syncExitFunc = os.Exit    // sync.go  
var doctorExitFunc = os.Exit  // doctor.go
```

### Preferred Pattern: Dependency Injection

```go
// ✅ Inject dependencies instead of using global seams
type CommandDeps struct {
    Clock      clock.Clock
    FileSystem FileSystemOps
    Logger     log.Logger
    Notify     func(bool, string) (bool, error)
}

type FileSystemOps struct {
    Stat      func(string) (fs.FileInfo, error)
    WriteFile func(string, []byte, fs.FileMode) error
    Remove    func(string) error
    MkdirAll  func(string, fs.FileMode) error
}

// In tests
deps := CommandDeps{
    Clock: clock.NewMock(),
    FileSystem: FileSystemOps{
        Stat: func(path string) (fs.FileInfo, error) {
            return MockFileInfo{}, nil
        },
    },
    Logger: testutil.NewTestLogger(t),
}
```

### Time-Based Testing

Use `benbjohnson/clock` for deterministic time testing:

```go
func TestDaemon_PeriodicSync(t *testing.T) {
    clk := clock.NewMock()
    deps := DaemonDeps{
        Clock: clk,
        Notify: func(_, _ string) (bool, error) { return true, nil },
    }
    
    // Start daemon in goroutine
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go daemon.Run(ctx, app, options, deps)
    
    // Advance time to trigger sync
    clk.Add(5 * time.Minute)
    
    // Verify sync occurred
    assert.Eventually(t, func() bool {
        return syncCallCount > 0
    }, time.Second, 10*time.Millisecond)
}
```

## Common Test Patterns

### 1. Validation Failure Test

Every command with PreRun validation should have this test:

```go
func TestCommand_ValidationFailure(t *testing.T) {
    app := NewAppBuilder(t).
        WithValidator(&MockValidator{
            SystemRequirementsFunc: func() error {
                return errors.New("validation failed")
            },
        }).
        Build(t)

    cmd := NewCommand().GetCobraCommand()
    SetupCommandContext(cmd, app)
    
    err := cmd.PreRunE(cmd, []string{})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
}
```

### 2. Successful Execution Test

Test the happy path with mocked dependencies:

```go
func TestCommand_Success(t *testing.T) {
    unitManager := &MockUnitManager{}
    app := NewAppBuilder(t).
        WithUnitManager(unitManager).
        Build(t)

    cmd := NewCommand().GetCobraCommand()
    SetupCommandContext(cmd, app)
    
    err := ExecuteCommand(t, cmd, []string{})
    assert.NoError(t, err)
    
    // Verify service calls
    assert.Len(t, unitManager.StartCalls, expectedCount)
}
```

### 3. Help Text Test

Test command help output:

```go
func TestCommand_Help(t *testing.T) {
    cmd := NewCommand().GetCobraCommand()
    output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})
    
    require.NoError(t, err)
    assert.Contains(t, output, "Expected help text")
}
```

### 4. Flag Testing

Test command-specific flags:

```go
func TestCommand_Flags(t *testing.T) {
    cmd := NewCommand().GetCobraCommand()
    
    flag := cmd.Flags().Lookup("my-flag")
    require.NotNil(t, flag)
    assert.Equal(t, "default-value", flag.DefValue)
}
```

### 5. Run Method with Mocked Dependencies

Test the Run method directly with full control over dependencies:

```go
func TestCommand_Run_SpecificScenario(t *testing.T) {
    app := NewAppBuilder(t).Build(t)
    cmd := NewCommand()
    
    // Create mock filesystem
    mockFS := &FileSystemOps{
        StatFunc: func(path string) (fs.FileInfo, error) {
            return MockFileInfo{name: "test", isDir: true}, nil
        },
        WriteFileFunc: func(_ string, _ []byte, _ fs.FileMode) error {
            return nil
        },
    }
    
    deps := CommandDeps{
        CommonDeps: CommonDeps{
            Clock:      clock.New(),
            FileSystem: mockFS,
            Logger:     testutil.NewTestLogger(t),
        },
        // Add command-specific mocks
    }
    
    err := cmd.Run(context.Background(), app, CommandOptions{}, deps)
    assert.NoError(t, err)
}
```

### 6. Skipping Complex Integration Tests

For tests that require complex setup better suited for integration testing:

```go
// TestCommand_ComplexScenario is skipped - requires full integration setup.
func TestCommand_ComplexScenario(t *testing.T) {
    t.Skip("Requires complex setup with git/compose - covered by integration tests")
}
```

## Mock Implementations

### Available Mocks

```go
// System validation
&MockValidator{
    SystemRequirementsFunc: func() error { return nil },
}

// Unit repository  
&MockUnitRepo{
    FindAllFunc: func() ([]repository.Unit, error) {
        return []repository.Unit{{Name: "test-unit"}}, nil
    },
}

// Unit manager (tracks calls)
&MockUnitManager{
    StartFunc: func(name, unitType string) error { return nil },
}
// Access recorded calls: mockManager.StartCalls, mockManager.StopCalls
```

### Creating Custom Mocks

When adding new dependencies to App, create corresponding mocks:

```go
type MockNewDependency struct {
    SomeMethodFunc func(param string) error
    CallHistory    []SomeCall
}

func (m *MockNewDependency) SomeMethod(param string) error {
    m.CallHistory = append(m.CallHistory, SomeCall{Param: param})
    if m.SomeMethodFunc != nil {
        return m.SomeMethodFunc(param)
    }
    return nil
}
```

## Adding New Command Tests

### 1. Create Test File

Create `cmd/newcommand_test.go` for `cmd/newcommand.go`:

```go
package cmd

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewCommand_BasicFunctionality(t *testing.T) {
    // Test implementation
}
```

### 2. Design Command with Dependency Injection

Structure your command to be testable:

```go
// In newcommand.go
type NewCommandOptions struct {
    Flag1 string
    Flag2 bool
}

type NewCommandDeps struct {
    FileSystem FileSystemOps
    Logger     log.Logger
    // Other dependencies
}

func (c *NewCommand) GetCobraCommand() *cobra.Command {
    var opts NewCommandOptions
    
    cmd := &cobra.Command{
        Use: "newcommand",
        PreRunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            return app.Validator.SystemRequirements()
        },
        RunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            deps := c.buildDeps(app)
            return c.Run(cmd.Context(), app, opts, deps)
        },
    }
    
    cmd.Flags().StringVar(&opts.Flag1, "flag1", "", "Description")
    cmd.Flags().BoolVar(&opts.Flag2, "flag2", false, "Description")
    return cmd
}

func (c *NewCommand) Run(ctx context.Context, app *App, opts NewCommandOptions, deps NewCommandDeps) error {
    // Testable implementation
    return nil
}
```

### 3. Test with Dependency Injection

```go
func TestNewCommand_Success(t *testing.T) {
    deps := NewCommandDeps{
        FileSystem: FileSystemOps{
            Stat: func(path string) (fs.FileInfo, error) {
                return MockFileInfo{}, nil
            },
        },
        Logger: testutil.NewTestLogger(t),
    }
    
    app := NewAppBuilder(t).Build(t)
    cmd := NewCommand{}
    
    err := cmd.Run(context.Background(), app, NewCommandOptions{}, deps)
    assert.NoError(t, err)
}
```

## Testing Best Practices

### DO:
- ✅ Test CLI behavior (flags, output, error codes)
- ✅ Use AppBuilder for consistent app construction
- ✅ Use test helpers for output capture
- ✅ Mock external dependencies via interfaces
- ✅ Use dependency injection instead of global seams
- ✅ Test both success and failure scenarios
- ✅ Follow naming convention: `TestCommandName_Scenario`
- ✅ Use `exec.CommandContext` when testing commands that need Cancel support
- ✅ Skip complex integration tests with clear rationale

### DON'T:
- ❌ Test internal business logic (that's tested elsewhere)
- ❌ Call real OS operations (systemd, git, file system)
- ❌ Use `os.Exit()` directly in commands (return errors instead)
- ❌ Use global mutable state for testing
- ❌ Test implementation details vs. user-facing behavior
- ❌ Use 0755 or higher for test directories (use 0750)
- ❌ Declare variables that won't be verified in assertions

## Example: Complete Command Implementation

```go
package cmd

import (
    "context"
    "fmt"
    "time"
    
    "github.com/benbjohnson/clock"
    "github.com/spf13/cobra"
)

// ExampleCommand implements the example command.
type ExampleCommand struct{}

// ExampleOptions holds command-line options.
type ExampleOptions struct {
    Interval time.Duration
    Force    bool
    Output   string
}

// ExampleDeps holds injected dependencies.
type ExampleDeps struct {
    Clock      clock.Clock
    FileSystem FileSystemOps
    Logger     log.Logger
}

func NewExampleCommand() *ExampleCommand {
    return &ExampleCommand{}
}

func (c *ExampleCommand) GetCobraCommand() *cobra.Command {
    var opts ExampleOptions
    
    cmd := &cobra.Command{
        Use:   "example",
        Short: "Example command with modern patterns",
        PreRunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            return app.Validator.SystemRequirements()
        },
        RunE: func(cmd *cobra.Command, _ []string) error {
            app := c.getApp(cmd)
            deps := c.buildDeps(app)
            return c.Run(cmd.Context(), app, opts, deps)
        },
        SilenceUsage:  true,
        SilenceErrors: true,
    }
    
    cmd.Flags().DurationVarP(&opts.Interval, "interval", "i", 5*time.Minute, "Operation interval")
    cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force operation")
    cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Output file")
    
    return cmd
}

func (c *ExampleCommand) Run(ctx context.Context, app *App, opts ExampleOptions, deps ExampleDeps) error {
    deps.Logger.Info("Starting example operation", "interval", opts.Interval)
    
    // Check if output file exists (unless forced)
    if opts.Output != "" && !opts.Force {
        if _, err := deps.FileSystem.Stat(opts.Output); err == nil {
            return fmt.Errorf("output file %s already exists (use --force to overwrite)", opts.Output)
        }
    }
    
    // Perform periodic operation
    ticker := deps.Clock.Ticker(opts.Interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            deps.Logger.Info("Operation cancelled")
            return ctx.Err()
        case <-ticker.C():
            if err := c.performOperation(ctx, opts, deps); err != nil {
                return fmt.Errorf("operation failed: %w", err)
            }
        }
    }
}

func (c *ExampleCommand) performOperation(ctx context.Context, opts ExampleOptions, deps ExampleDeps) error {
    // Implementation here
    deps.Logger.Debug("Performing operation")
    
    if opts.Output != "" {
        data := []byte("example output\n")
        if err := deps.FileSystem.WriteFile(opts.Output, data, 0644); err != nil {
            return fmt.Errorf("failed to write output: %w", err)
        }
    }
    
    return nil
}

func (c *ExampleCommand) buildDeps(app *App) ExampleDeps {
    return ExampleDeps{
        Clock:      clock.New(),
        FileSystem: FileSystemOps{
            Stat:      os.Stat,
            WriteFile: os.WriteFile,
            Remove:    os.Remove,
            MkdirAll:  os.MkdirAll,
        },
        Logger: app.Logger,
    }
}

func (c *ExampleCommand) getApp(cmd *cobra.Command) *App {
    return cmd.Context().Value(appContextKey).(*App)
}
```

## Migration Guide

### Migrating Existing Commands

To migrate legacy commands to the new pattern:

1. **Convert PreRun/Run to PreRunE/RunE**:
   ```go
   // ❌ Old pattern
   PreRun: func(cmd *cobra.Command, _ []string) {
       if err := validate(); err != nil {
           log.Error(err)
           os.Exit(1)
       }
   }
   
   // ✅ New pattern
   PreRunE: func(cmd *cobra.Command, _ []string) error {
       return validate()
   }
   ```

2. **Replace global seams with dependency injection**:
   ```go
   // ❌ Old pattern
   var exitFunc = os.Exit
   var osStat = os.Stat
   
   // ✅ New pattern
   type CommandDeps struct {
       FileSystem FileSystemOps
   }
   ```

3. **Move flags to options struct**:
   ```go
   // ❌ Old pattern
   var globalFlag string
   cmd.Flags().StringVar(&globalFlag, "flag", "", "Description")
   
   // ✅ New pattern
   type CommandOptions struct {
       Flag string
   }
   var opts CommandOptions
   cmd.Flags().StringVar(&opts.Flag, "flag", "", "Description")
   ```

## Troubleshooting

### Output Not Captured
- Commands using `fmt.Print*` require `ExecuteCommandWithCapture()`
- Commands using `cmd.Print*` work with standard Cobra `cmd.SetOut()`

### Dependency Injection Issues
- Ensure `buildDeps()` method provides real dependencies for production
- Verify dependency interfaces match expected signatures
- Check that all dependencies are properly initialized

### Context Cancellation
- Always check `ctx.Err()` early in long-running operations
- Pass context to all sub-operations that might block
- Use `ctx.Done()` in select statements for graceful shutdown

## Coverage Goals

Aim for these test scenarios per command:

1. **Validation failure** (if command has PreRun validation)
2. **Successful execution** (happy path)
3. **Error handling** (service failures, repository errors)
4. **Flag parsing** (command-specific flags)
5. **Help text** (verify help output)
6. **Output verification** (verbose vs non-verbose modes)

## Recent Test Additions (2025-09)

### Doctor Command (17 tests)
Comprehensive health check validation covering:
- All checks passing scenarios
- System requirements failures
- Missing/inaccessible config files
- Repository validation (not cloned, invalid git, missing compose dirs)
- Directory writability checks
- Structured output formats (JSON/YAML)
- Helper function validation

### Version Command (6 tests)
Version display and update checking:
- Version information output
- Development vs release version handling
- Update check behavior
- Build information display

### Validate Command (13 tests)
Docker Compose validation:
- Directory and single file validation
- Invalid compose files and paths
- Flag testing and mutual exclusivity
- System requirements checking
- Edge cases (empty dirs, non-YAML files)
- Helper function validation (isValidGitRepo, isComposeFile)

### Image Pull Command (12 tests)
Container image pulling:
- Verbose vs non-verbose modes
- User mode environment handling
- Error scenarios
- Dependency injection validation
- 2 complex tests marked as skipped for integration

This testing framework enables **comprehensive CLI testing without OS dependencies** while maintaining **fast, reliable, and maintainable tests**.

## Migration Summary

### Completed Migration Status

🎉 **All commands in the cmd package have been successfully migrated from global test seams to modern dependency injection patterns.**

**Migration Results:**
- **506+ tests** running in ~400ms with zero hanging tests
- **Zero global state** - All commands are stateless and thread-safe  
- **100% error-based flow** - No more `os.Exit` calls in command handlers
- **Deterministic testing** - Using `benbjohnson/clock` for reliable time-based tests
- **Comprehensive coverage** - Every command has validation, success, error, flags, and help tests

**Eliminated Legacy Patterns:**
- ❌ Global test seams (`exitFunc`, `syncExitFunc`, `doctorExitFunc`, etc.)
- ❌ Global flag variables (`repoName`, `force`, `dryRun`, etc.) 
- ❌ Direct `os.Exit()` calls in command handlers
- ❌ Mutable global state causing test interference
- ❌ Hanging or flaky tests due to infinite loops

**Established Modern Patterns:**
- ✅ `CommandOptions` structs for all flags
- ✅ `CommandDeps` structs with `CommonDeps` + command-specific dependencies
- ✅ `PreRunE/RunE` functions returning errors
- ✅ `Run(ctx, app, opts, deps)` method signatures for testability
- ✅ `buildDeps()` methods for production dependency construction
- ✅ Context-aware implementations with proper cancellation

This migration represents a **significant architectural improvement** that makes the codebase more maintainable, testable, and reliable while following modern Go best practices.
