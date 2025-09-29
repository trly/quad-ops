# CLI Testing Guide for @cmd Package

This guide explains how to structure CLI tests and work with the testing interfaces in the cmd package.

## Testing Architecture Overview

The cmd package uses **interface-based testing** with **dependency injection** to test CLI commands without requiring OS dependencies (systemd, podman, git, etc.).

### Core Principles

1. **Test User-Facing CLI Behavior** - Not internal business logic
2. **Mock OS Dependencies** - Use interfaces and test seams
3. **Use Cobra's Built-in Testing** - Leverage existing CLI testing patterns
4. **Follow Go Conventions** - Each `*.go` file has corresponding `*_test.go`

## Test File Organization

```
cmd/
├── up.go              ← Command implementation  
├── up_test.go         ← Tests for up command
├── sync.go            ← Command implementation
├── sync_test.go       ← Tests for sync command
├── mocks_test.go      ← Mock implementations (shared)
├── test_helpers.go    ← Test utilities (shared)
└── interfaces.go      ← Test interfaces (shared)
```

## Key Testing Components

### 1. SystemValidator Interface

All commands depend on system validation. Use the `SystemValidator` interface for testing:

```go
// In your test
app := NewAppBuilder(t).
    WithValidator(&MockValidator{
        SystemRequirementsFunc: func() error {
            return errors.New("systemd not found") // Simulate failure
        },
    }).
    Build(t)
```

### 2. AppBuilder Pattern

Use the fluent `AppBuilder` to construct test apps with mocked dependencies:

```go
app := NewAppBuilder(t).
    WithValidator(mockValidator).
    WithUnitRepo(mockRepo).
    WithUnitManager(mockManager).
    WithConfig(customConfig).
    WithVerbose(true).
    Build(t)
```

### 3. Exit Code Testing

Commands use `os.Exit()` which would kill the test process. Use the exit seam pattern:

```go
// Capture exit codes without killing test
var exitCode int
oldExit := exitFunc  // or syncExitFunc, doctorExitFunc, etc.
exitFunc = func(code int) { exitCode = code }
t.Cleanup(func() { exitFunc = oldExit })

// Execute command
cmd.PreRun(cmd, []string{})

// Verify exit behavior
assert.Equal(t, 1, exitCode)
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

Commands that interact with the OS use **function variable seams** that can be overridden in tests:

### Exit Functions
```go
// Each command has its own exit seam
var exitFunc = os.Exit        // up.go, down.go
var syncExitFunc = os.Exit    // sync.go  
var doctorExitFunc = os.Exit  // doctor.go
```

### File System Operations
```go
// doctor.go seams
var (
    osStat      = os.Stat
    osWriteFile = os.WriteFile
    osRemove    = os.Remove
)

// sync.go seams  
var (
    osMkdirAll = os.MkdirAll
    newGitRepository = git.NewGitRepository
    readProjects = compose.ReadProjects
)
```

### Using Seams in Tests

```go
func TestCommand_WithSeams(t *testing.T) {
    // Setup seams
    oldStat := osStat
    osStat = func(path string) (fs.FileInfo, error) {
        return MockFileInfo{}, nil  // Mock successful stat
    }
    t.Cleanup(func() { osStat = oldStat })
    
    // Run test...
}
```

## Common Test Patterns

### 1. Validation Failure Test

Every command with PreRun validation should have this test:

```go
func TestCommand_ValidationFailure(t *testing.T) {
    var exitCode int
    oldExit := exitFunc
    exitFunc = func(code int) { exitCode = code }
    t.Cleanup(func() { exitFunc = oldExit })

    app := NewAppBuilder(t).
        WithValidator(&MockValidator{
            SystemRequirementsFunc: func() error {
                return errors.New("systemd not found")
            },
        }).
        Build(t)

    cmd := NewCommand().GetCobraCommand()
    SetupCommandContext(cmd, app)
    cmd.PreRun(cmd, []string{})

    assert.Equal(t, 1, exitCode)
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

### 2. Add Exit Seams (if needed)

If your command calls `os.Exit()`, add a seam:

```go
// In newcommand.go
var newCommandExitFunc = os.Exit

// Replace os.Exit(1) with:
newCommandExitFunc(1)
```

### 3. Add OS Operation Seams (if needed)

If your command calls OS operations directly:

```go
// In newcommand.go  
var (
    osOperation = os.SomeOperation
    externalLib = external.SomeFunction
)

// Use in command:
if err := osOperation(param); err != nil {
    // handle error
}
```

### 4. Test Exit Behavior

```go
func TestNewCommand_ValidationFailure(t *testing.T) {
    var exitCode int
    oldExit := newCommandExitFunc
    newCommandExitFunc = func(code int) { exitCode = code }
    t.Cleanup(func() { newCommandExitFunc = oldExit })
    
    // Test that triggers exit
    assert.Equal(t, 1, exitCode)
}
```

## Testing Best Practices

### DO:
- ✅ Test CLI behavior (flags, output, exit codes)
- ✅ Use AppBuilder for consistent app construction
- ✅ Use test helpers for output capture
- ✅ Mock external dependencies via interfaces
- ✅ Use seams for OS operations and exit calls
- ✅ Test both success and failure scenarios
- ✅ Follow naming convention: `TestCommandName_Scenario`

### DON'T:
- ❌ Test internal business logic (that's tested elsewhere)
- ❌ Call real OS operations (systemd, git, file system)
- ❌ Use `os.Exit()` directly in commands (use seams)
- ❌ Forget to cleanup seams with `t.Cleanup()`
- ❌ Test implementation details vs. user-facing behavior

## Example: Complete Command Test

```go
package cmd

import (
    "errors"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestExampleCommand_ValidationFailure tests system requirements failure.
func TestExampleCommand_ValidationFailure(t *testing.T) {
    var exitCode int
    oldExit := exitFunc
    exitFunc = func(code int) { exitCode = code }
    t.Cleanup(func() { exitFunc = oldExit })

    app := NewAppBuilder(t).
        WithValidator(&MockValidator{
            SystemRequirementsFunc: func() error {
                return errors.New("validation failed")
            },
        }).
        Build(t)

    cmd := NewExampleCommand().GetCobraCommand()
    SetupCommandContext(cmd, app)
    cmd.PreRun(cmd, []string{})

    assert.Equal(t, 1, exitCode)
}

// TestExampleCommand_Success tests successful execution.
func TestExampleCommand_Success(t *testing.T) {
    app := NewAppBuilder(t).
        WithValidator(&MockValidator{}).
        Build(t)

    cmd := NewExampleCommand().GetCobraCommand()
    SetupCommandContext(cmd, app)

    output, err := ExecuteCommandWithCapture(t, cmd, []string{})
    
    require.NoError(t, err)
    assert.Contains(t, output, "Expected success message")
}

// TestExampleCommand_Help tests help output.
func TestExampleCommand_Help(t *testing.T) {
    cmd := NewExampleCommand().GetCobraCommand()
    output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})
    
    require.NoError(t, err)
    assert.Contains(t, output, "command description")
}
```

## Troubleshooting

### Output Not Captured
- Commands using `fmt.Print*` require `ExecuteCommandWithCapture()`
- Commands using `cmd.Print*` work with standard Cobra `cmd.SetOut()`

### Exit Code Not Captured
- Ensure command uses exit seam: `exitFunc(1)` not `os.Exit(1)`
- Check that seam variable is properly restored in `t.Cleanup()`

### Mock Interface Errors
- Ensure mock implements all interface methods
- Check method signatures match exactly (parameters and return types)
- Use existing mocks in `mocks_test.go` as reference

### Seam Function Not Working
- Verify seam variable is package-level in source file
- Ensure test calls seam, not original function directly
- Remember to restore original function in `t.Cleanup()`

## Coverage Goals

Aim for these test scenarios per command:

1. **Validation failure** (if command has PreRun validation)
2. **Successful execution** (happy path)
3. **Error handling** (service failures, repository errors)
4. **Flag parsing** (command-specific flags)
5. **Help text** (verify help output)
6. **Output verification** (verbose vs non-verbose modes)

This testing framework enables **comprehensive CLI testing without OS dependencies** while maintaining **fast, reliable, and maintainable tests**.
