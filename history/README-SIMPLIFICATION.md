# Pipeline Simplification Initiative

This directory contains comprehensive planning documentation for **simplifying the entire Compose→Spec→Quadlet/Launchd rendering pipeline**.

## The Challenge

The quad-ops codebase has grown to **12,432 lines of rendering code** spread across **33 files**:

- **internal/compose**: 1,800 lines (overly complex conversion)
- **internal/platform/systemd**: 5,345 lines (duplicated logic with launchd)
- **internal/platform/launchd**: 5,287 lines (duplicated logic with systemd)

This complexity makes the codebase hard to understand, maintain, and extend.

## The Solution

Reduce to **~2,750 lines** (78% reduction) through:
- Consolidating duplicate logic
- Removing dead abstractions
- Direct data mapping (Spec → output)
- Table-driven tests
- Single responsibility per file

**Guaranteed**: 100% behavior equivalence via golden tests

## Documentation Index

### 🚀 Start Here (5 minutes)
**[QUICK_REFERENCE.md](QUICK_REFERENCE.md)** - Executive summary with key metrics, phases, and effort estimate

### 🔍 Detailed Planning (30 minutes)
1. **[compose-simplification-plan.md](compose-simplification-plan.md)** - Compose package focus
   - Current bloat analysis (1800 lines)
   - Target structure (350 lines)
   - Method reduction strategy (26 → 12)
   - Expected benefits

2. **[full-pipeline-simplification.md](full-pipeline-simplification.md)** - Complete architecture
   - Full-stack overview
   - Systemd renderer simplification
   - Launchd renderer simplification
   - Shared podman args builder
   - Test consolidation strategy

### 💻 Implementation Guides
1. **[minimal-compose-reference.md](minimal-compose-reference.md)** - Code patterns and examples
   - Podman Quadlet spec mapping
   - What MUST convert vs what CAN'T
   - Minimal Converter structure
   - Testing strategy
   - Real code examples

2. **[implementation-checklist.md](implementation-checklist.md)** - Step-by-step execution
   - Detailed tasks for each phase
   - Validation checkpoints
   - Risk mitigation
   - Commit strategy
   - Success criteria

## The Complete Picture

### Phase-by-Phase Breakdown

```
Phase 1: Compose Simplification (4-6 hours)
├─ Write comprehensive tests (test-first)
├─ Consolidate methods (26 → 12)
├─ Delete unused interfaces
└─ All tests pass, coverage maintained

Phase 2: Shared Platform Logic (2-3 hours)
├─ Create platform/podman_args.go
├─ Both renderers use shared builder
├─ Eliminate 200+ lines duplication
└─ Output verified identical

Phase 3: Systemd Renderer (4-6 hours)
├─ Table-driven tests
├─ Direct Spec → INI mapping
├─ Remove intermediate models
└─ Units byte-for-byte equal

Phase 4: Launchd Renderer (4-6 hours)
├─ Table-driven tests
├─ Direct Spec → plist mapping
├─ Simplified options
└─ Plists byte-for-byte equal

Phase 5: Test Consolidation (2-3 hours)
├─ 16 test files → 8 focused files
├─ Table-driven format everywhere
├─ Coverage maintained or improved
└─ Tests become documentation

Phase 6-7: Integration & Docs (2-3 hours)
├─ Full test suite passes
├─ Golden tests validate equivalence
├─ Documentation updated
└─ Examples added
```

### Key Metrics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| **Total Lines** | 12,432 | 2,750 | -78% |
| **Compose** | 1,800 | 350 | -81% |
| **Systemd** | 5,345 | 1,200 | -78% |
| **Launchd** | 5,287 | 1,200 | -77% |
| **Test Files** | 16 | 8 | -50% |
| **Source Files** | 33 | ~20 | -39% |

## Key Principles

### Test-First Approach
- Write comprehensive tests BEFORE refactoring
- Tests become the spec for new behavior
- Golden tests ensure 100% behavior equivalence

### Direct Mapping
Instead of:
```
Spec → Model → Renderer → String
```

Use:
```
Spec → String (directly)
```

This is clearer, faster, and easier to debug.

### Single Responsibility
- Each file does one thing
- Each method solves one problem
- No dead code, no unused abstractions

### Zero Breaking Changes
- All public APIs unchanged
- CLI commands work identically
- Generated artifacts identical
- Configuration files unchanged

## Success Criteria

✅ **Functionality**
- All conversion tests pass
- All renderer tests pass
- Generated units identical
- Generated plists identical

✅ **Code Quality**
- 78% reduction in lines
- No dead code
- All linters pass
- Consistent formatting

✅ **Testing**
- Coverage maintained or improved
- Table-driven tests throughout
- Tests as documentation
- Easy to add new cases

✅ **Maintainability**
- Clear data flow
- Single responsibility
- Easy to understand (1-2 hour onboarding)
- Easy to extend (new platform = 1-2 files)

## How to Use This Documentation

### For Overview
1. Read QUICK_REFERENCE.md (5 min)
2. Review architecture diagram in full-pipeline-simplification.md

### For Planning
1. Read compose-simplification-plan.md
2. Read full-pipeline-simplification.md
3. Review metrics and timelines

### For Execution
1. Follow implementation-checklist.md phase-by-phase
2. Use minimal-compose-reference.md for code patterns
3. Run validation checkpoints after each phase
4. Commit incrementally with clear messages

### For Leadership
- QUICK_REFERENCE.md provides executive summary
- Metrics show ROI: 78% code reduction, zero breaking changes
- Risk mitigation strategies documented
- Phased approach allows incremental delivery

## Issue Tracking

**Main Issue**: [quad-ops-9f5](https://github.com/trly/quad-ops/issues) - Full pipeline simplification with acceptance criteria

Track progress by completing each phase:
- [ ] Phase 1: Compose simplification
- [ ] Phase 2: Shared podman args
- [ ] Phase 3: Systemd renderer
- [ ] Phase 4: Launchd renderer
- [ ] Phase 5: Test consolidation
- [ ] Phase 6-7: Integration & docs

## Questions?

These documents should answer:
- **What**: What are we simplifying and why?
- **How**: How do we do it step-by-step?
- **Why**: Why is this important and what are the benefits?
- **When**: How long will it take?
- **Risk**: What could go wrong and how do we prevent it?

If something is unclear, the issue tracker can be updated to clarify.

## Next Steps

1. **Read** QUICK_REFERENCE.md (5 minutes)
2. **Review** full-pipeline-simplification.md (30 minutes)
3. **Decide** whether to proceed with simplification
4. **Start** Phase 1 using implementation-checklist.md
5. **Report** progress on issue quad-ops-9f5

---

## Document Summary

| Document | Purpose | Read Time | Audience |
|----------|---------|-----------|----------|
| **QUICK_REFERENCE.md** | Executive summary | 5 min | Everyone |
| **compose-simplification-plan.md** | Compose package focus | 10 min | Developers |
| **full-pipeline-simplification.md** | Complete architecture | 20 min | Architects |
| **minimal-compose-reference.md** | Code patterns | 15 min | Developers |
| **implementation-checklist.md** | Step-by-step guide | 30 min | Implementers |

**Total Reading Time**: ~80 minutes for complete understanding  
**Core Reading Time**: ~15 minutes for high-level overview

---

Last Updated: November 11, 2025  
Created as part of quad-ops pipeline simplification initiative
