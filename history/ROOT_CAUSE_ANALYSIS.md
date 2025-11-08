# Root Cause Analysis: quad-ops-w2q

## Issue
Unit list is not finding platform-specific units that are managed by quad-ops on Linux systems.

**Reproduction:**
```bash
$ quad-ops sync -v --force  # Syncs and restarts 3 services successfully
$ quad-ops unit list -v     # Reports "No deployed artifacts found"
$ sudo podman ps -a         # Shows 13 running containers
```

The unit list command reports zero artifacts while containers are actively running and managed by quad-ops.

## Root Cause

**Location:** `cmd/app.go` lines 72-73, in the `NewApp()` function

**The Bug:**
The `RepoArtifactStore` is initialized with the wrong base directory:

```go
repoBaseDir := cfg.RepositoryDir  // ← WRONG: This is /var/lib/quad-ops
repoArtifactStore := repository.NewArtifactStore(fsService, logger, repoBaseDir)
```

**Why It Fails:**

1. `cfg.RepositoryDir` is where git repositories are cloned (e.g., `/var/lib/quad-ops/quad-ops-deploy/`)
2. Actual deployed artifacts are in `cfg.QuadletDir` (e.g., `/etc/containers/systemd/`)
3. The `unit list` command by default lists artifacts from `RepoArtifactStore`
4. When `RepoArtifactStore.List()` scans `/var/lib/quad-ops`, it finds the raw git clones (Docker Compose files, etc.), not the rendered `.container` and `.network` files
5. These rendered files only exist in `/etc/containers/systemd/` after the sync process

**Data Flow:**
- `sync` command: Reads Docker Compose from git repos → renders to `.container/.network/.volume` → writes to `/etc/containers/systemd/`
- `unit list` command (default): Attempts to list from `RepoArtifactStore` (baseDir=/var/lib/quad-ops) → finds Compose files, not rendered units → after filtering, reports "No deployed artifacts found"

**Why This Works for Sync:**
The `sync` command reads git-managed repositories and writes to the deployed directory, so it works correctly. The issue is only with `unit list` trying to list from the wrong directory.

## Technical Details

From `cmd/unit_list.go` lines 106-112:
```go
// Default: use git-managed artifacts from repository
deps.Logger.Debug("Listing artifacts from repository")
artifacts, err = deps.RepoArtifactStore.List(ctx)
if err != nil {
    return fmt.Errorf("failed to list repository artifacts: %w", err)
}
artifacts = filterArtifactsForPlatform(artifacts, app.Config)
```

The `RepoArtifactStore` scans `/var/lib/quad-ops` which contains git clones of repositories like `quad-ops-deploy`. After filtering by platform extensions (`.container`, `.network`), nothing matches because those rendered files are in `/etc/containers/systemd/`, not in the git repository directory.

The `--use-fs-artifacts` flag works correctly because it uses `ArtifactStore` (initialized with `cfg.QuadletDir`) instead.

## Solution

The `unit list` command should list artifacts from the **deployed** directory (`cfg.QuadletDir`), not the repository directory. 

**Fix Options:**

1. **Change default behavior (Recommended):** Make `unit list` default to using `deps.ArtifactStore` (deployed directory) instead of `deps.RepoArtifactStore` (repository directory). The current `--use-fs-artifacts` flag becomes redundant or is renamed for clarity.

2. **Clarify the names:** If `RepoArtifactStore` serves a different purpose (e.g., staging/processing), rename it and initialize it appropriately.

3. **Update initialization:** If `RepoArtifactStore` should represent processed artifacts, initialize it with a staging directory rather than the raw git repository directory.

The most pragmatic fix is Option 1: make the default behavior list from the deployed directory, as that's where the actual managed services are. The repository directory doesn't contain the rendered artifacts - only the Docker Compose source files that get processed into unit files.
