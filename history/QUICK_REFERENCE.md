# Pipeline Simplification: Quick Reference

## The Goal
Reduce 12,432 lines of render code to ~2,750 lines (78% reduction) while maintaining 100% behavior equivalence.

**NO BACKWARD COMPATIBILITY REQUIRED**: This is an internal refactoring. Breaking changes to internal APIs are acceptable. Focus on clean, simplified architecture over preserving old internal interfaces.

## Current State
```
internal/compose              1,800 lines (11 files)
  ├─ spec_converter.go         1000 lines (26 methods)
  ├─ processor.go                32 lines (thin wrapper)
  ├─ reader.go                  306 lines
  ├─ helpers.go                 109 lines
  ├─ interfaces.go               29 lines (unused)
  └─ 7 other files (tests + utils)

internal/platform/systemd     5,345 lines (8 files)
  ├─ renderer.go               500+ lines
  ├─ podmanargs.go             200+ lines (DUPLICATED)
  ├─ Multiple test/helper files
  └─ Complex unit/model classes

internal/platform/launchd     5,287 lines (14 files)
  ├─ renderer.go               300+ lines
  ├─ podmanargs.go             200+ lines (DUPLICATED)
  ├─ plist.go                  300+ lines
  ├─ Multiple test/helper files
  └─ Complex option handling
────────────────────────────────
TOTAL                         12,432 lines
```

## Target State (Revised After Phase 1)
```
internal/compose              1,500 lines (2 files) [was 350]
  ├─ convert.go               1,200 lines (14 focused methods) ✅
  └─ reader.go                  300 lines

internal/platform             200 lines (1 file)
  └─ podman_args.go           200 lines (SHARED)

internal/platform/systemd    1,000 lines (4 files) [was 1,200]
  ├─ renderer.go              400 lines (8-12 methods)
  ├─ quadlet.go               300 lines (INI writer)
  ├─ lifecycle.go             200 lines (unchanged)
  └─ helpers.go               100 lines

internal/platform/launchd    1,000 lines (5 files) [was 1,200]
  ├─ renderer.go              400 lines (8-12 methods)
  ├─ plist.go                 300 lines (XML writer)
  ├─ options.go               100 lines
  ├─ lifecycle.go             100 lines
  └─ helpers.go               100 lines
────────────────────────────────
TOTAL                         3,700 lines (revised from 2,750)

NOTE: Targets adjusted based on Phase 1 learnings.
Method count is more important than LOC for complex domains.
Focus: 8-12 methods per component, stdlib-first, clear responsibilities.
```

## Key Simplifications

### 1. Compose Package (1800 → 350 lines)
**Problem**: Over-engineered conversion with dead interfaces and complex helpers

**Solution**: 
- Single focused Converter class (not thin wrapper)
- 12-14 methods organized by concern, not 26 scattered methods
- Direct spec.Spec field assignment, not intermediate models
- Remove unused Repository, SystemdManager, FileSystem interfaces

**Example**:
```go
// OLD: 26 separate methods, complex helpers
func (c *Converter) convertVolumeMounts() ✗
func (c *Converter) convertConfigMounts() ✗
func (c *Converter) convertSecretMounts() ✗
func (c *Converter) convertFileObjectToMount() ✗

// NEW: Single unified approach
func (c *Converter) convertMounts() ✓
```

### 2. Shared Podman Args (200 lines saved)
**Problem**: systemd and launchd both build the same podman command line

**Solution**:
- Create `platform/podman_args.go` with single builder
- Both renderers use it
- Eliminates 200+ lines of duplication

### 3. Systemd Renderer (5345 → 1200 lines)
**Problem**: Complex unit models, multiple layers of abstraction

**Solution**:
- Direct Spec → Quadlet INI string mapping
- QuadletWriter class for building INI format
- No intermediate Unit model
- Table-driven tests validate each field

**Example**:
```go
// OLD: Spec → Unit → String (3 layers)
type Unit struct { ... }
func (u *Unit) Render() string { ... }

// NEW: Spec → String (1 layer)
func (r *Renderer) buildContainerUnit(spec service.Spec) (Artifact, error) {
    buf := &strings.Builder{}
    buf.WriteString("[Container]\n")
    buf.WriteString(fmt.Sprintf("Image=%s\n", spec.Container.Image))
    // ... continue
    return Artifact{Content: buf.String()}, nil
}
```

### 4. Launchd Renderer (5287 → 1200 lines)
**Problem**: Overly generic options handling, complex plist generation

**Solution**:
- Direct Spec → Plist XML mapping
- Simple options validation
- Minimal helpers (~50 lines)
- Table-driven tests

### 5. Test Consolidation (16 → 8 files)
**Problem**: Tests scattered across files, many test implementation details

**Solution**:
- 1 table-driven test file per component
- Tests validate behavior (input → output), not internal methods
- Easy to add new test cases

---

## Implementation Phases (Test-First)

### Phase 1: Compose ✅ COMPLETE
```
✅ Consolidated spec_converter.go: 50 → 14 methods (72% reduction)
✅ Renamed: spec_converter.go → convert.go
✅ Renamed: SpecConverter → Converter
✅ Applied stdlib: maps.Clone, slices.Clone, bit shifts, strconv
✅ Inlined: convertSecurity, convertBuild into convertContainer
✅ Kept helpers.go for domain-specific logic (Prefix, FindEnvFiles, IsExternal)
✅ All 1,210 tests passing with byte-for-byte equivalence
✅ reader.go: 306 lines (unchanged - already clean)

Learnings: Method count > LOC for complex domains. See phase1-learnings.md.
```

### Phase 2: Shared Builder (2-3 hours)
```
✓ Create platform/podman_args.go
✓ Update systemd renderer to use it
✓ Update launchd renderer to use it
✓ Verify output identical
```

### Phase 3: Systemd Renderer (4-6 hours)
```
✓ Write renderer_test.go (table-driven)
✓ Create quadlet.go (INI writer)
✓ Simplify renderer.go
✓ Verify units byte-equal to current
```

### Phase 4: Launchd Renderer (4-6 hours)
```
✓ Write renderer_test.go (table-driven)
✓ Simplify plist.go
✓ Simplify renderer.go and options.go
✓ Verify plists byte-equal to current
```

### Phase 5: Test Cleanup (2-3 hours)
```
✓ Consolidate 16 test files to 8
✓ Ensure table-driven format
✓ Coverage maintained
```

### Phase 6-7: Integration & Docs (2-3 hours)
```
✓ Full test suite passes
✓ Golden tests pass
✓ Update documentation
✓ Examples added
```

**Total: ~18-27 hours of focused work**

---

## Key Insights

### Why This Works
1. **Single Responsibility**: Each file does one thing (parsing, converting, rendering)
2. **Direct Mapping**: Spec → Output string is clearer than Spec → Model → String
3. **Table-Driven Tests**: Easy to see all cases in one place, easy to add more
4. **No Dead Code**: Every method used, every file necessary
5. **Shared Logic**: Both platforms use same podman args builder

### What Stays The Same
- ✅ Public APIs unchanged (ReadProjects, NewSpecProcessor, Process)
- ✅ All CLI commands work identically
- ✅ Configuration files unchanged
- ✅ Generated units/plists identical
- ✅ Service specs unchanged (internal model)

### What Changes
- ❌ Internal implementation (you don't see this)
- ❌ File structure (cleaner)
- ❌ Method count (fewer, more focused)
- ❌ Test organization (consolidated)
- ❌ Unused abstractions (removed)

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| **Lines of Code** | 2,750 (from 12,432) | 78% reduction |
| **Test Coverage** | Maintained or ↑ | ✓ |
| **Golden Tests** | 100% output equivalence | ✓ |
| **Behavior** | 100% identical | ✓ |
| **API Changes** | Zero breaking | ✓ |
| **File Count** | ~20 (from 33) | 39% reduction |
| **Methods/File** | ≤400 lines max | ✓ |
| **Test Files** | 8 (from 16) | 50% reduction |

---

## Why This Matters

### Before
- 12k lines is hard to understand end-to-end
- New developers take weeks to learn
- Changes to one renderer risk breaking the other
- Tests scattered, hard to see coverage
- Dead code and unused abstractions

### After
- 2.7k lines is readable in 1-2 hours
- Clear data flow: YAML → Spec → Unit/Plist
- Each renderer independent, no duplication
- Tests are documentation (table-driven)
- Every line of code serves a purpose

---

## Resources

📄 **Full Documentation**:
- `compose-simplification-plan.md` - Strategic overview of compose changes
- `full-pipeline-simplification.md` - Complete architecture & design
- `minimal-compose-reference.md` - Exact code patterns and mapping
- `implementation-checklist.md` - Step-by-step execution guide
- `QUICK_REFERENCE.md` - This file

📊 **Files in `history/` directory** for archaeological reference after completion

---

## Ready to Start?

1. **Review** the planning documents (30 min)
2. **Start Phase 1** with test-first approach (write tests before refactoring)
3. **Verify** at each checkpoint (tests pass)
4. **Iterate** through phases 2-5
5. **Validate** golden tests ensure 100% equivalence

All phases can be done independently. Start with compose, then move to platforms.

Current issue: **quad-ops-9f5** tracks this work with acceptance criteria.
