# quad-ops Agent Guidelines

GitOps framework for Podman containers on Linux and macOS.

## Commands

- **Build**: `task build` (fmt, lint, test, compile)
- **Test all**: `task test` or `gotestsum --format pkgname --format-icons text -- -coverprofile=coverage.out -v ./...`
- **Test single**: `go test -run TestName ./path/to/package -v`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

## Architecture

Core pipeline: `Docker Compose → Spec Converter → Platform Renderer → Lifecycle Manager`.

### Key Packages

- `internal/compose/` - Parses Docker Compose files, converts to platform-agnostic service specs
- `internal/platform/systemd/` - Quadlet unit file renderer for Linux
- `internal/platform/launchd/` - Plist renderer for macOS
- `internal/service/` - Core service specification models (models.go, validate.go)
- `internal/repository/` - Unit file storage and git synchronization
- `internal/systemd/` - systemd/DBus integration for lifecycle management
- `cmd/` - CLI commands using Cobra (sync, up, down, daemon, etc.)

## Compose Specification Scope

**quad-ops converts standard Docker Compose to Podman Quadlet units. NOT a Swarm orchestrator.**

### ✅ In Scope: Standard Compose + Podman Features

Support all container runtime features that work with standalone Podman:

- **Container basics**: image, build, command, entrypoint, working_dir, user, hostname
- **Environment**: environment, env_file, labels, annotations
- **Networking**: networks (bridge/host/custom), ports (host mode), dns\*, extra_hosts, network_mode
- **Storage**: volumes (bind/named/tmpfs), secrets/configs with **local sources only** (file/content/environment)
- **Resources**: memory, cpu (shares/quota/period), pids_limit, shm_size, sysctls, ulimits
- **Security**: cap_add/drop, privileged, security_opt, read_only, group_add, pid/ipc/cgroup modes
- **Devices**: devices, device_cgroup_rules, gpus (if Podman supports)
- **Health**: healthcheck, restart (maps to systemd), stop_signal, stop_grace_period
- **Dependencies**: depends_on (maps to systemd After/Requires, all conditions treated equally)

### ❌ Out of Scope: Docker Swarm Orchestration

**REJECT these features in validation with clear error messages:**

- **Multi-node orchestration**: deploy.placement, deploy.mode: global, deploy.replicas > 1
- **Rolling updates**: deploy.update_config, deploy.rollback_config
- **Service discovery**: deploy.endpoint_mode (vip/dnsrr)
- **Swarm networking**: ports with mode: ingress (use mode: host instead)
- **Swarm config/secrets store**: configs/secrets with `driver` field (use file/content sources)
- **Service labels**: deploy.labels (use top-level labels for containers)

**Error message format**: "Swarm orchestration not supported. Use Kubernetes/Nomad for feature X. Alternative: [workaround]"

### ⚠️ Key Distinctions

**Configs & Secrets** - Context matters:

- ✅ **Local sources** (file: ./config.txt, content: "data", environment: VAR) → Convert to bind mounts
- ❌ **Swarm store** (external: true with driver) → Reject with error

**Deploy section** - Mixed bag:

- ✅ deploy.resources.limits → Map to Quadlet resource directives
- ❌ deploy.placement/update_config/rollback_config → Swarm orchestration, reject

**Reference**: See history/v0.22.0-podman-only-scope.md for complete analysis

## Code Style

- **Testing**: table-driven tests preferred, heavy use of dependency injection and mocks
- **Imports**: Group stdlib, external packages, then internal (`github.com/trly/quad-ops/internal/*`)
- **Comments**: Package-level godoc required, exported functions documented
- **Error handling**: Wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Naming**: Service specs use sanitized names via `service.SanitizeName()`, unit files prefixed with project name
- **Validation**: All specs validated via `spec.Validate()` before rendering
- **Linters**: errcheck, govet, staticcheck, unused, revive, gosec, misspell enabled via golangci-lint

## Testing Libraries & Patterns

### Core Testing Stack

**Required libraries:**

- `testing` (stdlib) - Base test framework
- `github.com/stretchr/testify/assert` - Simple assertions
- `github.com/stretchr/testify/require` - Fatal assertions for preconditions
- `github.com/google/go-cmp/cmp` - Deep equality comparisons with diffs
- `go.uber.org/goleak` - Goroutine leak detection

### When to Use Each Library

#### Stdlib `testing`

**Use stdlib t.TempDir() for temporary directories:**

```go
func TestSomething(t *testing.T) {
    tempDir := t.TempDir()  // ✅ Auto-cleaned, simple
    // NOT: testutil.SetupTempDir(t)  // ❌ Don't use custom helpers
}
```

**Use t.Cleanup() for resource management:**

```go
func TestService(t *testing.T) {
    svc := NewService()
    t.Cleanup(func() {
        svc.Shutdown()
    })
}
```

#### testify (assert/require)

**Use testify for:**

- Simple assertions that don't need detailed diffs
- Error checking
- Boolean conditions
- Length/emptiness checks
- Contains/substring checks

```go
// ✅ Good testify usage
require.NoError(t, err)           // Fatal if error (stops test)
assert.Error(t, err)              // Non-fatal assertion
assert.True(t, condition)
assert.Len(t, slice, 3)
assert.Empty(t, result)
assert.Contains(t, output, "expected")
assert.Equal(t, 42, count)        // Simple scalar comparison
```

**Guideline**: Use `require` for setup/preconditions (failure should stop test). Use `assert` for actual test assertions (collect multiple failures).

#### go-cmp

**Use go-cmp for:**

- Struct comparisons
- Slice/map comparisons (especially when order matters)
- Complex nested data structures
- When you need to see **what** differs, not just **that** it differs

```go
import (
    "github.com/google/go-cmp/cmp"
    "github.com/google/go-cmp/cmp/cmpopts"
)

// ✅ Good go-cmp usage - shows detailed diff
want := []string{"db", "api", "web"}
if diff := cmp.Diff(want, got); diff != "" {
    t.Fatalf("start order mismatch (-want +got):\n%s", diff)
}

// ✅ Compare structs with detailed field-level diffs
expectedPod := &Pod{Name: "web", Image: "nginx"}
if diff := cmp.Diff(expectedPod, actualPod); diff != "" {
    t.Errorf("pod mismatch (-want +got):\n%s", diff)
}

// ✅ Ignore specific fields
if diff := cmp.Diff(want, got,
    cmpopts.IgnoreFields(Config{}, "Timestamp")); diff != "" {
    t.Errorf("config mismatch: %s", diff)
}

// ✅ Sort slices before comparing (order-independent)
if diff := cmp.Diff(want, got,
    cmpopts.SortSlices(func(a, b string) bool {
        return a < b
    })); diff != "" {
    t.Errorf("mismatch: %s", diff)
}
```

**When NOT to use go-cmp:**

- Simple scalar comparisons → use `assert.Equal`
- Error checking → use `assert.NoError` / `require.NoError`
- Boolean conditions → use `assert.True` / `assert.False`

#### goleak (Goroutine Leak Detection)

**Required for all packages that spawn goroutines.**

**Package-level detection (preferred):**

```go
// cmd/main_test.go or internal/daemon/main_test.go
package cmd

import (
    "testing"
    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

**Per-test detection (for complex daemon tests):**

```go
func TestDaemonLifecycle(t *testing.T) {
    defer goleak.VerifyNone(t)  // Verify no leaks at test end

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Ensure goroutines can exit cleanly
    done := make(chan struct{})
    go func() {
        daemon.Run(ctx)
        close(done)
    }()

    // ... test logic ...

    cancel()      // Signal shutdown
    <-done        // Wait for goroutine to exit
}
```

**Critical rules:**

- ✅ Always provide cancellation mechanism (context.WithCancel)
- ✅ Always wait for goroutines to complete before test ends
- ✅ Use channels to signal completion: `done := make(chan struct{})`
- ❌ Never spawn goroutines without synchronization
- ❌ Avoid `goleak.IgnoreAnyFunction` - fix the leak instead

**Ignoring known library goroutines (use sparingly):**

```go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m,
        // Document WHY each is ignored with issue link
        // https://github.com/example/lib/issues/123
        goleak.IgnoreTopFunction("example.com/lib.backgroundWorker"),
    )
}
```

#### Test Logging

**Use simple stdlib-based test logger (no custom handlers):**

```go
// cmd/test_helpers.go
func TestLogger(t *testing.T) log.Logger {
    var w io.Writer = io.Discard
    if testing.Verbose() {
        w = &testWriter{t}
    }
    slogLogger := slog.New(slog.NewTextHandler(w, nil))
    return log.NewSlogAdapter(slogLogger)
}

type testWriter struct{ t *testing.T }
func (w testWriter) Write(p []byte) (int, error) {
    w.t.Log(string(p))
    return len(p), nil
}

// Usage in tests
deps := DoctorDeps{
    CommonDeps: CommonDeps{
        Logger: TestLogger(t),  // ✅ Simple, stdlib-based
    },
}
```

**Guideline**: Don't assert on log output unless testing logging itself. Use simple discard/verbose pattern.

### Testing Anti-Patterns

❌ **DON'T create custom temp dir helpers** - use `t.TempDir()`
❌ **DON'T spawn goroutines without waiting** - causes flaky tests and goleak failures
❌ **DON'T use `assert.Equal` for complex structs** - use `cmp.Diff` for better error messages
❌ **DON'T mix assertion and precondition checks** - use `require` for preconditions
❌ **DON'T create elaborate test loggers** - simple discard/verbose is enough
❌ **DON'T use `time.Sleep` for synchronization** - use channels and context cancellation

### Testing Checklist

When writing tests:

1. ✅ Use `t.TempDir()` for temporary directories
2. ✅ Use `require.NoError(t, err)` for setup that must succeed
3. ✅ Use `assert.*` for test assertions
4. ✅ Use `cmp.Diff` for comparing structs, slices, maps
5. ✅ Add `TestMain` with `goleak.VerifyTestMain(m)` if package spawns goroutines
6. ✅ Ensure all goroutines exit cleanly (context cancellation + synchronization)
7. ✅ Use `t.Cleanup()` for resource teardown
8. ✅ Use TestLogger(t) for dependency injection (cmd tests)

**Reference issues:**

- quad-ops-uubw: t.TempDir migration
- quad-ops-we42: Simplified test logger
- quad-ops-mo35: goleak integration
- quad-ops-dqrd: go-cmp adoption

### Table-Driven Tests

Use table-driven tests for multiple test cases with the same testing logic. Follow [Go TableDrivenTests wiki](https://go.dev/wiki/TableDrivenTests) guidance.

**Pattern:**

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string      // Required: descriptive test case name
        input    InputType   // Test inputs
        expected OutputType  // Expected outputs
        wantErr  bool        // Optional: expect error
    }{
        {name: "basic case", input: ..., expected: ...},
        {name: "edge case", input: ..., expected: ...},
        {name: "error case", input: ..., wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic using tt.input, tt.expected
            got, err := DoSomething(tt.input)

            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

**When to use:**

- ✅ Multiple test cases with same testing logic
- ✅ Variations differ only in inputs/expected outputs
- ✅ You're copy-pasting test code with minor changes

**When NOT to use:**

- ❌ Test logic significantly different between cases
- ❌ Complex setup/teardown unique to specific scenarios
- ❌ Only 1-2 test cases

**Benefits:**

- Add test case = add row to table (not new function)
- Reduced code duplication through shared test logic
- Clear subtest names for debugging: `go test -run TestFeature/edge_case`
- Tests serve as documentation of all supported variations

## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" -t bug|feature|task -p 0-4 --json
bd create "Issue title" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`
6. **Commit together**: Always commit the `.beads/issues.jsonl` file together with the code changes so issue state stays in sync with code state

### Auto-Sync

bd automatically syncs with git:

- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### MCP Server (Recommended)

If using Claude or MCP-compatible clients, install the beads MCP server:

```bash
pip install beads-mcp
```

Add to MCP config (e.g., `~/.config/claude/config.json`):

```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

Then use `mcp__beads__*` functions instead of CLI commands.

### Managing AI-Generated Planning Documents

AI assistants often create planning and design documents during development:

- PLAN.md, IMPLEMENTATION.md, ARCHITECTURE.md
- DESIGN.md, CODEBASE_SUMMARY.md, INTEGRATION_PLAN.md
- TESTING_GUIDE.md, TECHNICAL_DESIGN.md, and similar files

**Best Practice: Use a dedicated directory for these ephemeral files**

**Recommended approach:**

- Create a `history/` directory in the project root
- Store ALL AI-generated planning/design docs in `history/`
- Keep the repository root clean and focused on permanent project files
- Only access `history/` when explicitly asked to review past planning

**Example .gitignore entry (optional):**

```
# AI planning documents (ephemeral)
history/
```

**Benefits:**

- ✅ Clean repository root
- ✅ Clear separation between ephemeral and permanent documentation
- ✅ Easy to exclude from version control if desired
- ✅ Preserves planning history for archeological research
- ✅ Reduces noise when browsing the project

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `history/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md and QUICKSTART.md.

### Landing the Plane

**When the user says "let's land the plane"**, follow this clean session-ending protocol:

1. **File beads issues for any remaining work** that needs follow-up
2. **Ensure all quality gates pass** (only if code changes were made) - run tests, linters, builds (file P0 issues if broken)
3. **Update beads issues** - close finished work, update status
4. **Sync the issue tracker carefully** - Work methodically to ensure both local and remote issues merge safely. This may require pulling, handling conflicts (sometimes accepting remote changes and re-importing), syncing the database, and verifying consistency. Be creative and patient - the goal is clean reconciliation where no issues are lost.
5. **Clean up git state** - Clear old stashes and prune dead remote branches:
   ```bash
   git stash clear                    # Remove old stashes
   git remote prune origin            # Clean up deleted remote branches
   ```
6. **Verify clean state** - Ensure all changes are committed and pushed, no untracked files remain
7. **Choose a follow-up issue for next session**
   - Provide a prompt for the user to give to you in the next session
   - Format: "Continue work on bd-X: [issue title]. [Brief context about what's been done and what's next]"

**Example "land the plane" session:**

```bash
# 1. File remaining work
bd create "Add integration tests for sync" -t task -p 2 --json

# 2. Run quality gates
task build

# 3. Close finished issues
bd close bd-42 bd-43 --reason "Completed" --json

# 4. Sync carefully - example workflow (adapt as needed):
git pull --rebase
# If conflicts in .beads/issues.jsonl, resolve thoughtfully:
#   - git checkout --theirs .beads/issues.jsonl (accept remote)
#   - bd import -i .beads/issues.jsonl (re-import)
#   - Or manual merge, then import
bd sync  # Export/import/verify
git push
# Repeat pull/push if needed until clean

# 5. Verify clean state
git status

# 6. Choose next work
bd ready --json
bd show bd-44 --json
```

**Then provide the user with:**

- Summary of what was completed this session
- What issues were filed for follow-up
- Status of quality gates (all passing / issues filed)
- Recommended prompt for next session
