# internal/buildinfo

Provides build metadata and update checking for quad-ops.

## Design

- `Version` is set via ldflags at build time by goreleaser
- `Commit`, `Date`, `GoVersion` are populated automatically from `runtime/debug.ReadBuildInfo()`
- `IsDev()` reports whether this is a development build (`Version == "dev"`)
- `CheckForUpdates()` queries GitHub releases via `go-selfupdate` and returns an `UpdateStatus`

## Conventions

- Only `Version` should use ldflags — all other metadata comes from the Go toolchain
- `CheckForUpdates` is context-aware and returns structured `UpdateStatus` rather than printing directly
- Keep update/version logic here; CLI output formatting belongs in `cmd/`
