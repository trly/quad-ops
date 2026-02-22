# internal/state

Manages deployment state persistence for quad-ops, tracking current and previous commit hashes per repository to enable rollback.

## Design

- State is stored as a JSON file on disk (`state.json`)
- `RepoState` tracks `current` and `previous` commit hashes per repository
- `SetCommit` is idempotent — setting the same hash twice does not shift `previous`
- `Load` returns an empty state (not an error) when the file does not exist
- `ManagedUnits` tracks quadlet unit filenames per repository via `SetManagedUnits`/`GetManagedUnits`, enabling stale resource cleanup — callers snapshot all managed units before and after sync, then diff to find units that were removed
- **Invariant: state must always reflect what is on disk.** `SetCommit` and `SetManagedUnits` must be called after unit generation regardless of partial failure, so that stale detection stays accurate
- Stale unit cleanup, state persistence, and daemon reload must always run — even on partial failure — so that successfully-synced repos stay consistent with their checked-out revision

## Conventions

- Keep the package focused on state persistence only — no business logic beyond commit tracking
- All filesystem errors must be wrapped with `fmt.Errorf("context: %w", err)`
- Tests use `testify/assert` and `testify/require`; test against the public API
