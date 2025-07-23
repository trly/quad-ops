# Agent Guidelines for git Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `git` package provides Git repository management functionality for quad-ops. It handles repository cloning, syncing, reference checkout, and local repository operations to support configuration management workflows.

## Key Structures and Interfaces

### Core Structures
- **`Repository`** - Main repository structure containing:
  - `config.Repository` - Embedded configuration
  - `Path` - Local filesystem path for the repository
  - `repo` - Internal go-git repository instance
  - `verbose` - Logging verbosity flag

### Key Functions
- **`NewGitRepository(repository config.Repository)`** - Creates a new Repository instance
- **`SyncRepository()`** - Clones or updates the repository
- **`checkoutTarget()`** - Checks out specific commit/branch/tag
- **`pullLatest()`** - Pulls latest changes from origin

### Key Dependencies
- **`github.com/go-git/go-git/v5`** - Git operations library
- **`github.com/go-git/go-git/v5/plumbing`** - Git plumbing types
- **`internal/config`** - Configuration access
- **`internal/log`** - Centralized logging

## Usage Patterns

### Repository Creation and Sync
```go
// Create repository instance
repo := git.NewGitRepository(configRepo)

// Sync (clone or update)
err := repo.SyncRepository()
if err != nil {
    return fmt.Errorf("failed to sync repository: %w", err)
}
```

### Repository Path Structure
- Local path pattern: `{RepositoryDir}/{repository.Name}`
- System mode: `/var/lib/quad-ops/{name}`
- User mode: `$HOME/.local/share/quad-ops/{name}`

## Development Guidelines

### Repository Operations
- **Clone**: Downloads repository if it doesn't exist locally
- **Update**: Pulls latest changes if repository already exists
- **Checkout**: Switches to specific commit, branch, or tag after sync

### Reference Handling
The package supports multiple reference types:
- **Commit hashes**: Direct SHA commit references
- **Branches**: Branch names (e.g., "main", "develop")
- **Tags**: Git tag references (e.g., "v1.0.0")

### Error Handling
- `git.ErrRepositoryAlreadyExists` is handled gracefully (not an error)
- `git.NoErrAlreadyUpToDate` during pulls is expected behavior
- Invalid references are reported as errors
- Network errors during clone/pull are propagated

### Logging Strategy
- Info level: Major operations (sync, clone, checkout)
- Debug level: Detailed operation progress
- Progress output directed to stdout during clone operations

## Common Patterns

### Safe Repository Sync
```go
func (r *Repository) SyncRepository() error {
    log.GetLogger().Info("Syncing repository", "path", filepath.Base(r.Path))
    
    repo, err := git.PlainClone(r.Path, false, &git.CloneOptions{
        URL:      r.URL,
        Progress: os.Stdout,
    })
    
    if err != nil {
        if err == git.ErrRepositoryAlreadyExists {
            // Repository exists, open and update it
            repo, err = git.PlainOpen(r.Path)
            if err != nil {
                return err
            }
            r.repo = repo
            return r.pullLatest()
        }
        return err
    }
    
    r.repo = repo
    if r.Reference != "" {
        return r.checkoutTarget()
    }
    return nil
}
```

### Flexible Reference Checkout
```go
func (r *Repository) checkoutTarget() error {
    worktree, err := r.repo.Worktree()
    if err != nil {
        return err
    }
    
    // Try as commit hash first
    hash := plumbing.NewHash(r.Reference)
    err = worktree.Checkout(&git.CheckoutOptions{Hash: hash})
    if err == nil {
        return nil
    }
    
    // Fall back to branch/tag
    return worktree.Checkout(&git.CheckoutOptions{
        Branch: plumbing.NewBranchReferenceName(r.Reference),
    })
}
```

## Git Operations Best Practices

### Clone Options
- Uses `PlainClone` for simple repository operations
- Progress output helps with large repository feedback
- No recursive submodule support (can be added if needed)

### Update Strategy
- Always attempts to pull from origin
- Handles "already up to date" as success case
- Does not force updates (respects local changes)

### Reference Resolution
- Attempts commit hash first (more specific)
- Falls back to branch name if hash fails
- Clear logging for debugging reference issues

## Configuration Integration

### Repository Configuration
```go
type Repository struct {
    Name       string // Repository identifier
    URL        string // Git clone URL
    Reference  string // Branch/tag/commit to checkout
    ComposeDir string // Directory within repo for compose files
}
```

### Path Management
- Local path automatically constructed from config
- Repository directory configurable per installation
- Supports both system-wide and user-specific installations

## Error Recovery

### Network Issues
- Clone failures are propagated to caller
- Pull failures (except "up to date") are errors
- No automatic retry logic (handled at higher level)

### Local Repository Issues
- Corrupted repositories cause re-clone
- Permission issues are reported clearly
- Missing directories are created automatically

## Performance Considerations

### Memory Usage
- Repository objects are held in memory during operations
- Large repositories may impact memory usage  
- Consider implementing repository object pooling for high-volume scenarios

### Disk Space
- Local repositories persist between syncs
- No automatic cleanup of old repositories
- Consider implementing cleanup policies for long-running deployments
