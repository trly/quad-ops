# Renderer.go Review: Sanitization & Unit Suffix Logic

## Key Issues & Refactoring Opportunities

### 1. **Unit Suffix Management - Fragmentation**

**Problem:** Unit type suffixes are hardcoded in multiple places:
- `formatDependency()` (lines 857-881): Lists 8 suffixes as a slice literal
- Artifact path construction (lines 78, 92, 105, 116): Hardcoded `.volume`, `.network`, `.build`, `.container`
- Dependency directives (lines 167-168, 205-209, 216-217): Hardcoded `.volume`, `.network`, `.build` suffixes

**Risk:** Adding/changing a unit type requires updates in 3+ places. Inconsistencies can emerge.

**Refactoring:** Extract to a constant package-level map or the Renderer type:
```go
const (
    UnitTypeContainer = ".container"
    UnitTypeNetwork   = ".network"
    UnitTypeVolume    = ".volume"
    UnitTypeBuild     = ".build"
    // ... others
)

// Or: registry/checker function
func hasUnitTypeSuffix(name string) bool { ... }
func appendUnitType(name, unitType string) string { ... }
```

---

### 2. **Network Name Sanitization - Complexity**

**Problem:** `resolveNetworkName()` (lines 939-961) handles an overly complex mapping:
- Strips `.network` suffix manually
- Tries exact match
- Falls back to `SanitizeName()` on the cleaned reference
- Tries matching again
- Returns sanitized form on no match

**Root Cause:** The compose parser should normalize network names ONCE during conversion. The renderer shouldn't need to "resolve" — it should receive already-resolved names.

**Refactoring:** 
- Verify the compose spec converter (`internal/compose/spec_converter.go`) is sanitizing all network names upfront
- Remove `resolveNetworkName()` from renderer entirely if spec.Networks are pre-sanitized
- If resolution is needed, move it to the spec converter layer (closer to the source)

**Current Usage:**
- Lines 190-191: Network dependency resolution
- Lines 422-423: Network directives in `addNetworks()`

---

### 3. **Redundant Suffix Stripping in `resolveNetworkName()`**

**Line 941:** `cleanRef := strings.TrimSuffix(ref, ".network")`

**Question:** Why does `ServiceNetworks` include `.network` suffix? This suggests:
- Either the compose converter is adding it (shouldn't)
- Or this is defensive programming against malformed input

**Recommendation:** Audit where `.network` suffixes come from in `ServiceNetworks`. If the compose converter is adding them, remove that logic there instead.

---

### 4. **Repetitive Slice Sorting Pattern**

**Appearances:** Lines 281, 370, 389, 407, 425, 534, 759-761, 798-800, 815-817, 824-826, 833-836, 842-845, etc.

```go
// Pattern repeats ~15+ times:
sorted := make([]string, len(items))
copy(sorted, items)
sort.Strings(sorted)
for _, item := range sorted {
    builder.WriteString(formatKeyValue("Key", item))
}
```

**Refactoring:** Extract a helper:
```go
func (r *Renderer) sortAndWrite(builder *strings.Builder, key string, items []string) {
    sorted := make([]string, len(items))
    copy(sorted, items)
    sort.Strings(sorted)
    for _, item := range sorted {
        builder.WriteString(formatKeyValue(key, item))
    }
}
```

**Benefit:** DRY principle, easier to update sorting behavior globally.

---

### 5. **`formatDependency()` - Inefficient Suffix Checking**

**Lines 873-877:** Linear search through suffix slice on every call
```go
for _, suffix := range suffixes {
    if strings.HasSuffix(dep, suffix) {
        return dep
    }
}
```

**Refactoring:** Use a map for O(1) reverse lookup:
```go
var knownSuffixes = map[string]bool{
    ".network": true,
    ".volume":  true,
    // ...
}

func (r *Renderer) hasUnitTypeSuffix(name string) bool {
    for suffix := range knownSuffixes {
        if strings.HasSuffix(name, suffix) {
            return true
        }
    }
    return false
}
```

Or use a regexp cached at package init if suffix patterns become more complex.

---

### 6. **Inconsistent Dependency Addition Pattern**

**Observation:** Three code paths add dependencies (lines 150-161, 164-171, 196-212):

```go
// DependsOn
for _, dep := range deps {
    depUnit := r.formatDependency(dep)
    builder.WriteString(fmt.Sprintf("After=%s\n", depUnit))
    builder.WriteString(fmt.Sprintf("Requires=%s\n", depUnit))
}

// Volumes
for _, vol := range spec.Volumes {
    builder.WriteString(fmt.Sprintf("After=%s.volume\n", vol.Name))
    builder.WriteString(fmt.Sprintf("Requires=%s.volume\n", vol.Name))
}

// Networks
for _, net := range networks {
    builder.WriteString(fmt.Sprintf("After=%s.network\n", net))
    if !externalNetworks[net] {
        builder.WriteString(fmt.Sprintf("Requires=%s.network\n", net))
    }
}
```

**Refactoring:** Extract a helper:
```go
func (r *Renderer) addDependency(builder *strings.Builder, unitName string, hard bool) {
    builder.WriteString(fmt.Sprintf("After=%s\n", unitName))
    if hard {
        builder.WriteString(fmt.Sprintf("Requires=%s\n", unitName))
    }
}
```

Then call it consistently with `r.addDependency(builder, "name.type", isRequired)`.

---

### 7. **Network Sanitization Happens Too Late**

**Current flow (lines 186-212):**
1. Get `ServiceNetworks` from container (may be unsanitized)
2. Resolve each via `resolveNetworkName()`
3. Add After/Requires directives
4. Add Network directives (lines 418-429) — resolves AGAIN

**Better flow:**
- Sanitize once in compose converter
- Renderer assumes all names are pre-sanitized
- No resolution needed

---

## Summary Table

| Issue | Severity | Effort | Impact |
|-------|----------|--------|--------|
| Unit suffix fragmentation | Medium | Low | Maintainability, consistency |
| Complex network resolution | Medium | Medium | Reduces complexity, removes duplication |
| Repetitive slice sorting | Low | Low | Code clarity, DRY |
| `formatDependency()` efficiency | Low | Trivial | Micro-optimization (negligible) |
| Inconsistent dep addition | Low | Low | Code clarity, consistency |
| Late sanitization | Medium | High | Architectural improvement |

## Recommended Priority

1. **Extract unit type constants** (Low effort, high clarity)
2. **Consolidate dependency addition** into a helper (Low effort, high clarity)
3. **Move network sanitization earlier** (compose converter) (Medium effort, cleaner arch)
4. **Evaluate `resolveNetworkName()` necessity** (Medium effort, may simplify significantly)
5. **Extract slice-sort helper** (Low effort, DRY)

## Quick Wins

- Move unit type suffixes to package-level constants
- Add `addDependency()` helper to reduce duplication
- Cache sorted network/volume names to avoid repeat sorting in unit vs. directive sections
