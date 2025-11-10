# Plan: Remove Underscore-to-Hyphen Replacement (quad-ops-x67)

## Problem Statement

The current `SanitizeName()` function in `internal/service/validate.go` performs regex-based replacement of invalid characters with hyphens. This adds complexity and potential for drift between network definition and service reference code paths.

**Current behavior:**
```go
result := regexp.MustCompile(`[^a-zA-Z0-9_.-]+`).ReplaceAllString(name, "-")
```

This replaces ANY character not in `[a-zA-Z0-9_.-]` with a hyphen, effectively:
- Converting spaces to hyphens
- Converting any special chars to hyphens  
- BUT preserving underscores, hyphens, and dots

The validation regex (`serviceNameRegex`) already allows `[a-zA-Z0-9_.-]*`, so sanitization should be unnecessary if input is pre-validated.

## Root Cause

**Why underscore-to-hyphen replacement exists:**
- Compose files may contain names with spaces/special chars (e.g., "my app", "my-service")
- systemd unit names require alphanumeric + hyphen/underscore/dot
- Current code tries to normalize these to safe names

**Why it's problematic:**
1. **Unnecessary complexity**: If input names are already validated, no replacement needed
2. **Consistency risk**: Two code paths (`convertProjectNetworks` vs `convertNetworkMode`) must apply identical logic
3. **Not a true replacement**: The function only replaces invalid chars with `-`, not underscores to hyphens
4. **False sense of safety**: Validation happens AFTER conversion, not before

## Simplification Strategy

### Phase 1: Understand Current Usage (Analysis) ✅ COMPLETED

**Question 1:** Are incoming names from Compose already validated before `SanitizeName()` is called?
- **Answer:** ✅ Yes. Names come from `types.Project` (compose-spec parser), which validates them per Docker Compose spec.

**Question 2:** What names actually need sanitization?
- Container names: `SanitizeName(Prefix(project.Name, serviceName))` 
- Volume names: `SanitizeName(Prefix(project.Name, volumeName))`
- Network names: `SanitizeName(resolvedName)` or `SanitizeName(Prefix(...))`

**Question 3:** Can `Prefix()` function produce invalid characters?
- **Answer:** ✅ No. `Prefix()` only concatenates with hyphen: `fmt.Sprintf("%s-%s", projectName, resourceName)`
- If both inputs are valid (already from compose-spec validation), output is valid
- Project names and resource names are both alphanumeric with optional hyphens/underscores/dots
- Result maintains that character set

### Phase 2: Proposed Simplification

**Option A: Remove `SanitizeName()` entirely**
- Rely on compose-spec validation (names are already validated)
- Rely on `Prefix()` function to produce valid output
- Requires audit: ensure ALL name inputs are pre-validated

**Option B: Replace with no-op or simple validation**
```go
func SanitizeName(name string) string {
    // Only validate format, don't transform
    // If name is invalid, caller should validate before calling this
    return name
}
```

**Option C: Make `SanitizeName()` a true validator + transformer**
```go
func SanitizeName(name string) (string, error) {
    // Validate: must match serviceNameRegex
    if !serviceNameRegex.MatchString(name) {
        return "", fmt.Errorf("invalid name %q", name)
    }
    return name, nil
}
```

### Phase 3: Implementation Path

**Recommended approach: Option A (Remove entirely)**

**Steps:**
1. **Audit**: Verify all names passed to `SanitizeName()` are already validated
   - Service names: prefixed from validated service name
   - Network names: from compose-spec parser or explicit declarations
   - Volume names: from compose-spec parser
   
2. **Replace `SanitizeName()` calls with direct name usage**
   - Remove the function call wrapper
   - Update tests to assert direct names (not sanitized)
   - Example: `service.SanitizeName("my-network")` → `"my-network"`

3. **Add validation layer earlier**
   - Validate names when converting from Compose format
   - Fail fast if invalid, don't try to fix
   - Follows "fail noisily" principle (Unix Philosophy)

4. **Update tests**
   - Remove `service.SanitizeName()` calls from assertions
   - Assert on raw names
   - Tests become simpler and more direct

5. **Remove `SanitizeName()` function**
   - Delete from `internal/service/validate.go`
   - Update comments that reference it

### Phase 4: Benefits

- **Simpler code**: No regex-based transformation logic needed
- **Consistency guaranteed**: No two-path divergence possible
- **Fail-fast**: Invalid names caught early with clear errors
- **Tests simpler**: Direct assertions instead of sanitization calls
- **Unix Philosophy compliant**: "Clarity is better than cleverness"

### Phase 5: Risk Assessment

**Risks:**
- ⚠️ May break backward compatibility if users have workflows that rely on sanitization
- ⚠️ Compose files with invalid names will fail conversion (vs being "fixed")

**Mitigations:**
- Add comprehensive validation error messages
- Document that names must be alphanumeric + hyphen/underscore/dot
- File issues for any discovered edge cases

## Code Locations to Check

1. `internal/service/validate.go` (lines 226-245): `SanitizeName()` implementation
2. `internal/service/models.go`: Name field definitions and their validation
3. `internal/compose/spec_converter.go`: All calls to `SanitizeName()` (grep shows ~12 calls)
4. `internal/compose/spec_converter_test.go`: Test assertions using `SanitizeName()`
5. `internal/compose/network_dependencies_test.go`: Network tests

## Testing Strategy

**Create new test suite: `sanitization_simplification_test.go`**
- Test that name validation happens at conversion time
- Test that invalid names are rejected with clear errors
- Test that valid names pass through unchanged
- Test prefixing logic doesn't introduce invalid characters
- Cross-check network definitions match service references

**Refactor existing tests:**
- Update `spec_converter_test.go` assertions to remove `service.SanitizeName()` calls
- Update `network_dependencies_test.go` likewise
- Add negative tests for invalid names

## Success Criteria

✅ All instances of `SanitizeName()` removed from codebase
✅ Tests updated and passing
✅ No functional behavior change for valid inputs
✅ Invalid inputs now fail with clear error messages (not silently fixed)
✅ Network name consistency maintained (tests prove it)
✅ Code coverage maintained or improved
