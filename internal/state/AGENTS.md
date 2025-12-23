# internal/state

Manages deployment state persistence for quad-ops, tracking current and previous commit hashes per repository to enable rollback.

## Design

- State is stored as a JSON file on disk (`state.json`)
- `RepoState` tracks `current` and `previous` commit hashes per repository
- `SetCommit` is idempotent — setting the same hash twice does not shift `previous`
- `Load` returns an empty state (not an error) when the file does not exist

## Conventions

- Keep the package focused on state persistence only — no business logic beyond commit tracking
- All filesystem errors must be wrapped with `fmt.Errorf("context: %w", err)`
- Tests use `testify/assert` and `testify/require`; test against the public API
