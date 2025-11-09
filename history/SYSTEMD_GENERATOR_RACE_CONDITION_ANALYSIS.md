# systemd Quadlet Generator Race Condition Analysis

**Date**: November 8, 2025
**Issue**: Services fail to restart after `quad-ops sync up` with "Unit not found" errors
**Severity**: High (production impact)
**Status**: Identified - Issues created for implementation

## Problem Summary

When running `quad-ops sync up -f`, services fail to restart with errors:

```
Failed to restart service llm-open-webui: error restarting unit llm-open-webui.service: Unit llm-open-webui.service not found.
Failed to restart service llm-ollama: error restarting unit llm-ollama.service: Unit llm-ollama.service not found.
Failed to restart service media-immich-server: error restarting unit media-immich-server.service: Unit media-immich-server.service not found.
```

However, the `.container` files **ARE** successfully written to `/etc/containers/systemd/`:

```
-rw-r--r--. 1 root root  414 Nov  8 21:10 llm-ollama.container
-rw-r--r--. 1 root root  487 Nov  8 21:10 llm-open-webui.container
-rw-r--r--. 1 root root  837 Nov  8 21:10 media-immich-server.container
```

## Root Cause: Asynchronous Reload Race Condition

The issue is a **race condition between systemd's Quadlet generator and quad-ops' restart operation**:

### The Problem

The D-Bus `Reload()` method in `internal/platform/systemd/lifecycle.go` is **asynchronous** and doesn't wait for the Quadlet generator to finish.

### Current Code Sequence

**File**: `cmd/sync.go` lines 168-175

```go
if anyChanges || opts.Force {
    deps.Logger.Info("Reloading service manager")
    if err := deps.Lifecycle.Reload(ctx); err != nil {
        return fmt.Errorf("failed to reload service manager: %w", err)
    }

    if len(servicesToRestart) > 0 {
        names := c.sortedServiceNames(servicesToRestart)
        restartErrs := deps.Lifecycle.RestartMany(ctx, names)  // ← This fails!
        // ...
    }
}
```

### Execution Timeline

1. ✅ **T+0ms**: Write `.container` files to `/etc/containers/systemd/`
   - Files are on disk: `llm-ollama.container`, `llm-open-webui.container`, etc.

2. ⚠️ **T+10ms**: Call `Lifecycle.Reload()`
   - D-Bus method: `manager.Reload()` 
   - Returns immediately (asynchronous)
   - Signals systemd to reload configuration
   - systemd spawns Quadlet generator in background

3. ❌ **T+15ms**: Call `Lifecycle.RestartMany()`
   - Tries to restart: `systemctl restart llm-open-webui.service`
   - But the `.service` unit doesn't exist yet!
   - Quadlet generator still processing `.container` files
   - Result: "Unit not found" error

### Why It's Intermittent

- **Fast systems**: Generator finishes before restart attempts → works fine
- **Slow systems**: Generator still running → race condition occurs
- **Large deployments**: More files to process → higher chance of timeout

## Technical Details

### How systemd Quadlet Generator Works

1. `systemctl daemon-reload` is called (or D-Bus Reload())
2. systemd invokes all system generators: `/usr/lib/systemd/system-generators/`
3. Podman's Quadlet generator is one of these:
   - Location: `/usr/lib/systemd/system-generators/podman-system-generator`
   - Reads all `.container`, `.pod`, `.network`, `.volume`, `.build` files from `/etc/containers/systemd/`
   - Converts them to `.service` units via systemd's dependency system
4. Generated units are placed in systemd's runtime directory
5. systemd parses and loads the new units

### The Synchronization Issue

The D-Bus `Reload()` call doesn't wait for generator completion:

**File**: `internal/systemd/dbus_connection.go` line 79-85

```go
func (d *DBusConnection) Reload(ctx context.Context) error {
    err := d.conn.ReloadContext(ctx)  // ← Async! No wait!
    if err != nil {
        return fmt.Errorf("error reloading systemd: %w", err)
    }
    return nil
}
```

The `ReloadContext()` from `coreos/go-systemd` is a simple async call. It doesn't wait for:
- Generators to finish
- Units to be loaded
- Dependencies to be resolved

## Evidence from Production System

The bdyvp22 system shows clear timing:

```
[root@bdyvp22 ~]# ls -al /etc/containers/systemd/ | grep -E "llm-|media-immich"
-rw-r--r--. 1 root root  414 Nov  8 21:10 llm-ollama.container
-rw-r--r--. 1 root root  487 Nov  8 21:10 llm-open-webui.container
-rw-r--r--. 1 root root  838 Nov  8 21:10 media-immich-server.container
```

File timestamps show they were written at 21:10, same time as the error messages:

```
time=2025-11-08T21:10:49.904-05:00 level=ERROR msg="Failed to restart service" name=llm-open-webui
time=2025-11-08T21:10:49.909-05:00 level=ERROR msg="Failed to restart service" name=llm-ollama
time=2025-11-08T21:10:49.911-05:00 level=ERROR msg="Failed to restart service" name=media-immich-server
```

This is a clear race condition: files written, reload called, restart attempted all within milliseconds.

## Secondary Issues

### Issue 2: Potential Generator Installation Problems

If the generator isn't installed or fails silently:
- `.container` files are written ✅
- `daemon-reload` completes without error ✅
- `.service` units are never generated ❌
- Users get "Unit not found" with no diagnostic info ❌

**Check**: Is `/usr/lib/systemd/system-generators/podman-system-generator` installed?

### Issue 3: Partial Updates

When syncing multiple compose files, some may update while others don't:
- Files timestamps: `21:05` (old), `21:10` (new)
- Dependency resolution might be incomplete if generator processes mixed states
- This could cause partial failures

## Solutions Implemented

### Issue quad-ops-22e: Race Condition Fix (P0 bug)
**Acceptance Criteria**: 
- Implement `waitForUnitGeneration()` with exponential backoff retry
- Verify units exist before calling RestartMany()
- Configurable timeout (default 5 seconds)
- Clear error messages if units don't appear

**Implementation Path**:
```go
// In lifecycle.go RestartMany():
for name := range servicesToRestart {
    if err := l.waitForUnitGeneration(ctx, name); err != nil {
        // Log diagnostic info and fail fast
        return err
    }
}

// New method:
func (l *Lifecycle) waitForUnitGeneration(ctx context.Context, serviceName string) error {
    timeout := 5 * time.Second
    pollInterval := 50 * time.Millisecond
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        // Try to get unit properties
        if _, err := l.unitManager.GetUnitProperties(ctx, serviceName+".service"); err == nil {
            return nil  // Unit exists!
        }
        
        select {
        case <-time.After(pollInterval):
            pollInterval = time.Duration(float64(pollInterval) * 1.5)  // Exponential backoff
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return fmt.Errorf("unit %s not generated after %v seconds", serviceName, timeout)
}
```

### Issue quad-ops-3zb: Diagnostic Tooling (P1 task)
**Acceptance Criteria**:
- Detect if generator is installed
- Compare artifact files vs systemd units
- Better error messages: "Unit file written but generator didn't create service"
- Optional: `quad-ops debug systemd` helper command

### Issue quad-ops-yx7: Documentation (P1 task)
**Acceptance Criteria**:
- Document Quadlet generator installation requirements
- Explain systemd/Podman version compatibility
- Troubleshooting guide for "Unit not found" errors
- User vs system mode setup differences

### Issue quad-ops-6sz: Unit Existence Verification (P1 task)
**Acceptance Criteria**:
- Add retry logic with exponential backoff
- Make timeout configurable
- Log progress for debugging

## Testing Strategy

### Unit Tests
1. Test `waitForUnitGeneration()` with immediate success
2. Test retry logic with eventual success
3. Test timeout behavior
4. Test context cancellation

### Integration Tests
1. Write `.container` file and verify `waitForUnitGeneration()` waits
2. Write multiple files and verify all units are generated
3. Verify timeout error is clear and diagnostic

### Manual Testing
1. Deploy to slow system with large compose files
2. Verify restart succeeds even with generator delay
3. Manually remove generator and verify clear error

## Workaround for Users (Immediate)

Until the fix is deployed, users experiencing this issue can:

1. **Manual restart after sync**:
   ```bash
   quad-ops sync
   sleep 2  # Give generator time
   systemctl restart llm-ollama llm-open-webui media-immich-server
   ```

2. **Run two separate commands**:
   ```bash
   quad-ops sync  # Write files only
   sleep 2
   quad-ops up -f  # Restart services
   ```

3. **Check generator status**:
   ```bash
   # Verify generator is installed
   ls -la /usr/lib/systemd/system-generators/ | grep podman
   
   # Check systemd logs for generator errors
   journalctl -u systemd-system-generators.target -n 50
   ```

## Long-term Improvements

1. **Configuration**: Make retry timeout user-configurable in `config.yaml`
2. **Health Checks**: Periodic verification that generator is installed and working
3. **Metrics**: Log timing information for generator completion
4. **Documentation**: Clear requirements for Quadlet generator availability

## Files Affected

| File | Change | Reason |
|------|--------|--------|
| `cmd/sync.go` | Add unit existence check after Reload() | Fix race condition |
| `internal/platform/systemd/lifecycle.go` | Add `waitForUnitGeneration()` method | Implement retry logic |
| `internal/systemd/dbus_connection.go` | Consider adding explicit wait option | Future improvement |
| `docs/INSTALLATION.md` | Document generator requirements | User education |
| Internal tests | Add test coverage for retry logic | Quality assurance |

## Related Issues

- **Memory/CPU resources not rendered** (quad-ops-24q) - Data loss issue
- **Duplicate PidsLimit** (quad-ops-ld0) - Validation issue
- **TimeoutStartSec missing** (quad-ops-494) - Production readiness
- **Entrypoint escaping** (quad-ops-ni3) - Edge case handling
- **Network dependencies** (quad-ops-kgm) - Unnecessary coupling

---

**Status**: Documented and issues created. Awaiting implementation.
