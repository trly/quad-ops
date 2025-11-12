# Internal/Compose Simplification Plan

## Analysis Summary

The `internal/compose` package currently serves one purpose: **Convert Docker Compose specifications to platform-agnostic service.Spec models for rendering as Quadlet units**.

### Current Code Bloat

**11 source files + 8 test files:**

```
spec_converter.go         ~1000 lines  - Main conversion logic (26 public methods)
processor.go              32 lines     - Thin wrapper (2 methods)
reader.go                 306 lines    - Compose file parsing + validation
helpers.go                109 lines    - Name utilities, env discovery
interfaces.go             29 lines     - 3 unused interfaces (Repository, SystemdManager, FileSystem)
namespace_modes_test.go   ~200 lines   - Test utilities
network_dependencies_test.go ~100 lines - Test utilities
sysctls_test.go           ~100 lines   - Test utilities
volume_dependencies_test.go ~100 lines - Test utilities
```

### What Actually Gets Used

From tracing imports across the codebase:
- **Used API**: 
  - `compose.ReadProjects()` - Parse compose files
  - `compose.NewSpecProcessor()` - Create processor
  - `processor.Process(ctx, project)` - Convert to service.Spec

- **Unused**:
  - `Repository` interface - Never implemented or used
  - `SystemdManager` interface - Platform-specific, doesn't belong in compose pkg
  - `FileSystem` interface - Never implemented
  - `HasNamingConflict()` - Dead code
  - `convertInitVolumeMounts()` - Only used internally
  - Various test helper utilities duplicated across test files

### Issues with Current Design

1. **Over-abstraction**: Interfaces for Repository, SystemdManager, FileSystem serve no purpose
2. **Redundant wrapper**: SpecProcessor just wraps SpecConverter with one line
3. **Complex helpers**: Env file discovery and name resolution are simple functions blown up
4. **Split concerns**: Tests are scattered across 8 files with duplicated setup
5. **Method explosion**: SpecConverter has 26+ public/private methods with unclear boundaries
6. **File object complexity**: convertFileObjectToMount() and related is 60+ lines for simple bind mount logic

## Proposed Simplification

### Target Structure (400-500 lines total)

```
convert.go           ~350 lines - Single SpecConverter with focused methods
reader.go           ~150 lines - ReadProjects() and ParseComposeFile() only  
helpers.go           ~30 lines - Prefix(), SanitizeProjectName(), IsExternal()
convert_test.go      ~400 lines - Comprehensive spec converter tests
reader_test.go       ~200 lines - Reader parsing tests
```

### Key Refactoring Steps

#### 1. **Remove Unused Interfaces** 
- Delete `interfaces.go` entirely
- Repository, SystemdManager, FileSystem belong in their respective packages
- Passes interfaces to converter should be removed (they're never passed anyway)

#### 2. **Inline SpecProcessor**
- Merge processor.go into convert.go as a NewConverter() factory
- Current code: `NewSpecProcessor()` → `NewConverter()`
- Single responsibility: convert.go handles Compose→Spec transformation

#### 3. **Consolidate SpecConverter Methods**
Group by concern, reduce from 26 to ~12 key methods:

```go
// Public API
func NewConverter(workingDir string) *Converter
func (c *Converter) Convert(project *types.Project) ([]service.Spec, error)

// Private helpers grouped by concern
// Container conversion (11 methods → 3)
convertContainer()
convertSecurity()
convertResources()

// Network/Volume conversion (7 methods → 2)  
convertVolumeMounts()
convertNetworks()

// Config/Secret handling (5 methods → 1)
convertFileObject()  // handles configs, secrets, mounts uniformly

// Misc conversions (3 methods → 2)
convertPorts()
convertHealthcheck()
```

#### 4. **Simplify File Object Handling**
Current: 3 separate methods (convertConfigMounts, convertSecretMounts, convertFileObjectToMount)
Proposed: Single unified approach for local sources:

```go
// All local sources (file, content, environment) → bind mounts
convertFileObject(obj FileObjectConfig, ...) Mount
```

#### 5. **Inline Simple Helpers**
- `Prefix()` - 3 lines → inline in convert.go
- `SanitizeProjectName()` - 4 lines → inline in reader.go  
- `NameResolver()` - 4 lines → inline where used
- Keep only: `IsExternal()` and `FindEnvFiles()` as they're used multiple times

#### 6. **Consolidate Tests**
Merge 8 test files into 2:

**convert_test.go** (replaces spec_converter_test.go + golden_test.go):
- Table-driven spec conversion tests
- Golden file assertions
- Extension handling (x-quad-ops-init, x-podman-env-secrets)

**reader_test.go** (replaces reader_test.go + helpers_test.go):
- Project parsing tests
- Environment file discovery
- Project name sanitization

Delete test files that test internal mechanics:
- namespace_modes_test.go - Tests internal convertNetworkMode() 
- network_dependencies_test.go - Tests internal network logic
- sysctls_test.go - Tests internal sysctl conversion
- volume_dependencies_test.go - Tests internal volume logic
- helpers_test.go - Tests internal helpers

These are all implementation details; test through public API instead.

### Method Reduction Target

| Concern | Current | Proposed | Simplification |
|---------|---------|----------|-----------------|
| Container | 8 methods | 3 methods | Combine similar conversions |
| Network/Volume | 8 methods | 2 methods | Consolidate mount handling |
| Config/Secrets | 5 methods | 1 method | Unified FileObject approach |
| Resources/Health | 8 methods | 2 methods | Direct mapping where possible |
| **Total** | **26+** | **12-14** | **Reduce by 50%** |

## Implementation Order

1. **Phase 1 - Tests First**
   - Write comprehensive convert_test.go with table-driven approach
   - Tests become the spec for the new API

2. **Phase 2 - Consolidate Convert**
   - Merge helpers.go into convert.go
   - Inline simple utilities
   - Remove unused interfaces

3. **Phase 3 - Simplify Methods**
   - Group by concern
   - Reduce method count by consolidating similar logic
   - Keep internal structure but reduce public surface

4. **Phase 4 - Clean Up Tests**
   - Delete implementation-detail test files
   - Consolidate into convert_test.go and reader_test.go

5. **Phase 5 - Verify**
   - All existing golden tests pass
   - No coverage reduction
   - Package API unchanged from caller perspective

## Expected Benefits

- **Less code**: 1800+ lines → 500 lines (72% reduction)
- **Easier maintenance**: Single focused file vs distributed across 11
- **Clearer intent**: Conversion logic in one place, easy to trace
- **Better testing**: Table-driven tests easier to read and extend
- **No behavior change**: All conversions remain identical

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Break existing API | Keep ReadProjects(), NewSpecProcessor(), Process() signatures |
| Lose test coverage | Run full test suite before/after; measure coverage |
| Introduce bugs | Refactor method-by-method; run integration tests between phases |
| Miss edge cases | Golden test ensures output equivalence |

## Success Criteria

✅ All conversion tests pass (golden test, edge cases)  
✅ Coverage maintained or improved  
✅ Code reduced to ~500 lines (from 1800+)  
✅ No external API changes  
✅ Clear separation: parsing (reader.go) vs conversion (convert.go)
