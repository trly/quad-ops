# Table-Driven Test Consolidation

**Issue**: quad-ops-6  
**Date**: 2025-11-11  
**Objective**: Consolidate scattered tests into table-driven patterns following Go best practices

## Motivation

The codebase had fragmented test files with many separate test functions testing related scenarios. This made it:
- Hard to see what was tested and what was missing
- Difficult to add new test cases
- Verbose with repeated test setup code
- Unclear which variations were covered

## Approach

Following [Go TableDrivenTests wiki](https://go.dev/wiki/TableDrivenTests) guidance:
1. Group related test scenarios into single table-driven tests
2. Use `t.Run()` for clear subtest reporting
3. Reduce code duplication through shared test logic
4. Make tests serve as documentation

## Changes Made

### systemd renderer (internal/platform/systemd/renderer_test.go)

**Before**: 15 test functions  
**After**: 7 test functions (53% reduction)

#### Consolidations:

1. **RequiresMountsFor tests** (4 → 1)
   - `TestRequiresMountsFor_MixedMountTypes`
   - `TestRequiresMountsFor_ReadOnlyBindMount`
   - `TestRequiresMountsFor_EmptySourcePath`
   - `TestRequiresMountsFor_RelativePathBindMount`
   
   **Became**: `TestRequiresMountsFor` with 4 table-driven cases

2. **NetworkOnlineTarget tests** (4 → 1)
   - `TestNetworkOnlineTarget_ContainerWithNetworksAndPorts`
   - `TestNetworkOnlineTarget_ContainerWithoutNetworksOrPorts`
   - `TestNetworkOnlineTarget_ContainerWithHostNetwork`
   - `TestNetworkOnlineTarget_InitContainer`
   
   **Became**: `TestNetworkOnlineTarget` with 4 table-driven cases

3. **Sysctls tests** (3 → 1)
   - `TestRenderer_Sysctls`
   - `TestRenderer_NoSysctls`
   - `TestRenderer_SysctlsFormat`
   
   **Enhanced**: `TestRenderer_Sysctls` with 4 table-driven cases (added "no sysctls" case)

### compose converter (internal/compose/convert_test.go)

**Before**: 29 test functions  
**After**: 27 test functions (2 removed)

#### Consolidations:

1. **Sysctls tests** (3 → 1)
   - `TestConverter_Sysctls`
   - `TestConverter_NoSysctls`
   - `TestConverter_EmptySysctls`
   
   **Became**: `TestConverter_Sysctls` with 5 table-driven cases

## Table-Driven Pattern Example

```go
func TestRequiresMountsFor(t *testing.T) {
    tests := []struct {
        name            string
        mounts          []service.Mount
        wantContains    []string
        wantNotContains []string
    }{
        {
            name: "mixed mount types",
            mounts: []service.Mount{
                {Type: service.MountTypeBind, Source: "/host/data", Target: "/app/data"},
                {Type: service.MountTypeVolume, Source: "my-volume", Target: "/app/vol"},
                {Type: service.MountTypeTmpfs, Target: "/tmp"},
            },
            wantContains: []string{"RequiresMountsFor=/host/data"},
            wantNotContains: []string{
                "RequiresMountsFor=my-volume",
                "RequiresMountsFor=/tmp",
            },
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Shared test logic
            logger := testutil.NewTestLogger(t)
            r := NewRenderer(logger)
            
            spec := service.Spec{
                Name: "app",
                Container: service.Container{
                    Image: "alpine:latest",
                    Mounts: tt.mounts,
                },
            }
            
            result, err := r.Render(context.Background(), []service.Spec{spec})
            require.NoError(t, err)
            
            // Assertions based on test case expectations
            for _, want := range tt.wantContains {
                assert.Contains(t, result, want)
            }
            for _, notWant := range tt.wantNotContains {
                assert.NotContains(t, result, notWant)
            }
        })
    }
}
```

## Benefits Achieved

### Readability
- Clear test case names describe what's being tested
- Easy to scan all variations in one place
- Tests serve as documentation of behavior

### Maintainability
- Adding new test case = adding row to table (no new test function)
- Shared test logic reduces duplication
- Consistent structure across all tests

### Debugging
- `t.Run()` provides clear subtest reporting
- Failed tests show exact case name
- Easy to run specific subtests: `go test -run TestRequiresMountsFor/mixed_mount_types`

### Coverage
- Maintained 67.0% coverage (no reduction)
- All 932 tests pass
- Behavior validation unchanged

## Key Patterns

### Test Structure
```go
tests := []struct {
    name     string        // Required: descriptive test case name
    input    InputType     // Test inputs
    expected ExpectedType  // Expected outputs
    wantErr  bool          // Optional: expect error
}{
    {name: "case1", input: ..., expected: ...},
    {name: "case2", input: ..., expected: ...},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic using tt.input, tt.expected
    })
}
```

### When to Use Table-Driven Tests
✅ **Use when**:
- Multiple test cases follow same testing logic
- Variations differ only in inputs/expected outputs
- You find yourself copy-pasting test code

❌ **Don't use when**:
- Test logic is significantly different between cases
- Complex setup/teardown unique to specific scenarios
- Only 1-2 test cases

## Future Opportunities

### Additional consolidation candidates:

1. **NetworkDependencies tests** (6 separate tests)
   - Could become 1 table-driven test with 6 cases

2. **VolumeDependencies tests** (6 separate tests)
   - Could become 1 table-driven test with 6 cases

3. **Healthcheck tests** (2 separate tests)
   - Could become 1 table-driven test with 2 cases

### Potential improvements:
- Add `t.Parallel()` to table-driven tests for faster execution
- Apply pattern to launchd tests
- Consider golden test pattern for complex output validation

## Metrics

### Test Function Reduction
- **systemd**: 15 → 7 functions (8 removed, 53% reduction)
- **compose**: 29 → 27 functions (2 removed, 7% reduction)
- **Total**: 44 → 34 test functions (10 removed, 23% reduction)

### Test Coverage
- **Before**: 67.0%
- **After**: 67.0% (maintained)

### Test Count
- **Total tests**: 932 (unchanged - consolidated functions now have subtests)

## References

- [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [Go Testing Best Practices](https://go.dev/doc/effective_go#testing)
- Issue: quad-ops-6
