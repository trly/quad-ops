# git Package

Git repository management for quad-ops using go-git.

## Purpose

Provides clone, sync, and checkout operations for remote repositories containing Docker Compose files.

## Key Types

- `Repository` - Represents a Git repository with URL, reference (branch/tag/commit), and local path

## Public API

- `New(name, url, ref, composeDir, path)` - Create a Repository instance
- `(*Repository).Sync(ctx)` - Clone or pull latest changes
- `(*Repository).GetCurrentCommitHash()` - Return current HEAD commit hash

## Dependencies

- `github.com/go-git/go-git/v5` - Pure Go git implementation

## Testing

Tests use local git repositories created with `git.PlainInit`. Use `createTestRepo` helper to set up test fixtures with initial commits.

```bash
go test -v ./internal/git/...
```

## Code Conventions

- Context parameter accepted but not yet used for cancellation
- Private methods: `checkoutTarget`, `pullLatest`
- Errors wrapped with `fmt.Errorf` for context
- `git.NoErrAlreadyUpToDate` treated as success in pull operations
