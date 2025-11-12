# Stdlib Helper Replacement - Phase 1 Enhancement

**Date**: 2025-11-11  
**Issue**: quad-ops-2 (Phase 1: Compose simplification)  
**Scope**: Replace custom helpers with Go stdlib equivalents

## Overview

As part of Phase 1 compose simplification, we consulted the Oracle and Librarian to identify opportunities to replace custom helper methods with Go standard library capabilities or well-maintained packages.

## Changes Implemented

### 1. ✅ Replaced `copyStringMap` with `maps.Clone()`

**Before**:
```go
func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
```

**After**:
```go
import "maps"

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return maps.Clone(in)
}
```

**Impact**: Uses Go 1.21+ stdlib, cleaner implementation, maintains nil-on-empty semantics

### 2. ✅ Replaced `internal/sorting` with `sort.Strings()`

**Before**:
```go
import "github.com/trly/quad-ops/internal/sorting"

sorting.SortStringSlice(deps)
sorting.SortStringSlice(dns)
sorting.SortStringSlice(dnsSearch)
// ... 8 call sites total
```

**After**:
```go
import "sort"

sort.Strings(deps)
sort.Strings(dns)
sort.Strings(dnsSearch)
// ... 8 call sites total
```

**Impact**: 
- Eliminated dependency on internal/sorting package
- Uses stdlib sort.Strings directly
- 8 call sites updated

### 3. ✅ Deleted `NameResolver` helper, inlined simple coalesce pattern

**Before**:
```go
func NameResolver(explicitName, defaultName string) string {
	if explicitName != "" {
		return explicitName
	}
	return defaultName
}

resolvedName := NameResolver(projectNet.Name, networkName) // 7 call sites
```

**After**:
```go
resolvedName := networkName
if projectNet.Name != "" {
	resolvedName = projectNet.Name
}
```

**Impact**: 
- Deleted 1 trivial helper function
- Inlined at 7 call sites with explicit if-else pattern
- More readable, no function call overhead

### 4. ✅ Inlined `get*FromMap` helpers at call sites

**Before**:
```go
func getStringFromMap(m map[string]interface{}, key string) string { ... }
func getMapFromMap(m map[string]interface{}, key string) map[string]string { ... }
func getStringSliceFromMap(m map[string]interface{}, key string) []string { ... }

initEnv := getMapFromMap(initMap, "environment")
initVolumes := getStringSliceFromMap(initMap, "volumes")
image := getStringFromMap(initMap, "image")
command := getStringSliceFromMap(initMap, "command")
```

**After**:
```go
// Inlined directly in convertInitContainers (only usage site)
// Extract image (string)
var image string
if v, ok := initMap["image"].(string); ok {
	image = v
}

// Extract command (string or []interface{})
var command []string
if v, ok := initMap["command"]; ok {
	switch cmd := v.(type) {
	case []interface{}:
		for _, item := range cmd {
			if s, ok := item.(string); ok {
				command = append(command, s)
			}
		}
	case string:
		command = []string{cmd}
	}
}
// ... similar for environment and volumes
```

**Impact**:
- Deleted 3 generic helper functions (60+ lines)
- Inlined at only call site (convertInitContainers)
- More explicit type handling, better error visibility

### 5. ✅ Optimized `SanitizeProjectName` - precompiled regexes

**Before**:
```go
func SanitizeProjectName(name string) string {
	normalized := strings.ReplaceAll(name, " ", "-")
	normalized = regexp.MustCompile(`^[^a-zA-Z0-9]+|[^a-zA-Z0-9]+$`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`-+`).ReplaceAllString(normalized, "-")
	return normalized
}
```

**After**:
```go
var (
	leadingTrailingNonAlphanumericRegex = regexp.MustCompile(`^[^a-zA-Z0-9]+|[^a-zA-Z0-9]+$`)
	multipleHyphensRegex                = regexp.MustCompile(`-+`)
)

func SanitizeProjectName(name string) string {
	normalized := strings.ReplaceAll(name, " ", "-")
	normalized = leadingTrailingNonAlphanumericRegex.ReplaceAllString(normalized, "")
	normalized = multipleHyphensRegex.ReplaceAllString(normalized, "-")
	return normalized
}
```

**Impact**:
- Regexes compiled once at package init, not on every call
- Performance improvement for high-frequency sanitization
- No behavior change

## Helpers Kept (Domain-Specific)

These helpers remain as they have no stdlib equivalent and are domain-specific to quad-ops:

1. **`Prefix(projectName, resourceName)`** - Prefixing with deduplication logic
2. **`FindEnvFiles(serviceName, workingDir)`** - Multi-pattern env file discovery
3. **`HasNamingConflict(repo, unitName, unitType)`** - Unit name conflict detection
4. **`IsExternal(external interface{})`** - Handle types.External and various bool types
5. **`formatBytes(bytes types.UnitBytes)`** - Human-readable byte formatting

### Considered but Deferred: `formatBytes`

Oracle/Librarian recommended `github.com/docker/go-units`, but:
- Already in dependency tree (transitive via compose-go)
- Our custom formatBytes is 10 lines, well-tested
- Would need to verify output format compatibility
- **Deferred** to avoid introducing subtle formatting differences

## Alignment with QUICK_REFERENCE.md

These changes directly support **Phase 1: Compose (1800 → 350 lines)** goals:

✅ Merge helpers inline  
✅ Use stdlib over custom implementations  
✅ Remove dead abstractions  
✅ Reduce helper count

## Metrics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| **Helper functions in spec_converter.go** | 5 (copyStringMap + 3 get*FromMap + formatBytes) | 2 (copyStringMap + formatBytes) | -3 |
| **Helper functions in helpers.go** | 5 | 4 | -1 (NameResolver) |
| **External package dependencies** | internal/sorting | stdlib only (maps, sort) | -1 internal dep |
| **Stdlib usage** | Limited | maps.Clone, sort.Strings | +2 stdlib |
| **Lines of helper code** | ~110 | ~50 | -60 lines |

## Test Coverage

**All tests pass** with no regressions:
- ✅ Golden tests (byte-for-byte output equivalence)
- ✅ Unit tests (helpers_test.go, spec_converter_test.go)
- ✅ Integration tests

## Next Steps

Continuing Phase 1 consolidation:
1. Inline convertCPU + convertCPUShares into convertResources
2. Consolidate ensureProjectTempDir + writeTempFile + convertFileObjectToMount
3. Review convertEnvFiles for potential inlining

**Target**: 21 methods → 12-15 methods

## References

- Oracle analysis: Reviewed stdlib coverage and trade-offs
- Librarian analysis: Explored Docker/Moby patterns for map copying, byte formatting, type assertions
- Go 1.25 stdlib docs: maps.Clone, sort.Strings
- QUICK_REFERENCE.md Phase 1 goals
