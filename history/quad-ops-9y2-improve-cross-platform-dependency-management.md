# quad-ops-9y2: Improve Cross-Platform Dependency Management

## Issue Summary

The current dependency system works well for systemd but has gaps for launchd (macOS). Launchd platform lacks proper dependency ordering in lifecycle operations and doesn't express dependencies in generated plist files.

## Current State Analysis

### Systemd (Linux)
- ✅ Dependencies rendered as `After` and `Requires` directives in unit files
- ✅ `StartMany` starts services concurrently (dependencies handled by systemd)
- ✅ `StopMany` stops in reverse order of provided list
- ✅ Dependency validation exists in `internal/service/validate.go`
- ✅ Dependency graph management in `internal/dependency/`

### Launchd (macOS)
- ❌ No dependency information in generated plist files
- ❌ `StartMany` iterates services without respecting order
- ❌ `StopMany` iterates services without reverse dependency order
- ❌ `RestartMany` iterates services without respecting order

### Shared Infrastructure
- ✅ Dependency graph construction in `cmd/up.go`
- ✅ Service ordering via `orderAndExpand()` method
- ✅ Dependency validation during spec conversion
- ✅ `DependsOn` field in service specs

## Required Work Breakdown

### 1. Implement Dependency Ordering in Launchd Lifecycle

**File**: `internal/platform/launchd/lifecycle.go`

**Changes**:
- Modify `StartMany()` to process services sequentially in the provided order
- Modify `StopMany()` to process services in reverse order (like systemd)
- Modify `RestartMany()` to process services in the provided order
- Add logging to indicate sequential processing for dependencies

**Rationale**: Since launchd plists don't contain dependency metadata, ordering must be enforced at the lifecycle level.

### 2. Add Dependency Validation During Spec Conversion

**File**: `internal/service/validate.go` (extend existing validation)

**Changes**:
- Add cycle detection validation using the dependency graph
- Validate that all `DependsOn` services exist
- Add platform-specific dependency validation (e.g., launchd limits)

**Rationale**: Currently validation only checks basic service references, but doesn't validate the dependency graph integrity.

### 3. Improve Dependency Resolution and Cycle Detection

**File**: `internal/dependency/dependency.go`

**Changes**:
- Add cycle detection algorithm to `ServiceDependencyGraph`
- Add methods for dependency resolution and conflict detection
- Improve error messages for dependency issues
- Add unit tests for cycle detection

**Rationale**: The current graph supports building but lacks cycle detection, which is critical for dependency management.

### 4. Add Platform-Specific Dependency Mapping

**Files**:
- `internal/platform/systemd/renderer.go` (enhance existing)
- `internal/platform/launchd/renderer.go` (add dependency support)
- `internal/platform/launchd/plist.go` (add dependency fields to Plist struct)

**Changes**:
- **Systemd**: Ensure comprehensive `After`/`Requires` mapping
- **Launchd**: Add plist fields for dependency expression (if supported by launchd)
  - Research launchd dependency mechanisms (QueueDirectories, WatchPaths, etc.)
  - Add dependency-related fields to Plist struct
  - Modify renderer to include dependency information

**Rationale**: Systemd expresses dependencies in unit files; launchd should express them in plists where possible.

### 5. Test Dependency Handling Across Platforms

**Files**:
- `cmd/up_test.go` (extend existing tests)
- `internal/platform/launchd/lifecycle_test.go` (add dependency ordering tests)
- Integration tests for multi-service scenarios

**Changes**:
- Add tests verifying services start/stop in correct dependency order on both platforms
- Add tests for dependency cycle detection
- Add integration tests with real compose files containing dependencies
- Test platform-specific behavior differences

**Rationale**: Ensure dependency management works correctly and consistently across Linux and macOS.

## Implementation Strategy

1. **Phase 1**: Fix launchd lifecycle ordering (items 1)
2. **Phase 2**: Add dependency validation and cycle detection (items 2, 3)
3. **Phase 3**: Platform-specific dependency mapping (item 4)
4. **Phase 4**: Comprehensive testing (item 5)

## Success Criteria

- Services start/stop in correct dependency order on both platforms
- Dependency cycles are detected and reported during validation
- Launchd respects dependency ordering even without plist-level dependencies
- All existing functionality preserved
- Test coverage maintained/improved

## Risk Assessment

- **Low Risk**: Lifecycle ordering changes (sequential processing may be slightly slower but more reliable)
- **Medium Risk**: Adding dependency fields to launchd plist (may require research into launchd capabilities)
- **Low Risk**: Enhanced validation (additive changes)
- **Low Risk**: Additional tests (additive changes)

## Dependencies

None - this is a self-contained improvement to existing dependency infrastructure.
