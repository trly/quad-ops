# cmd Package - Agent Guidelines

CLI entrypoint for quad-ops using the Kong CLI framework.

## Structure

- `quad-ops/main.go` - Application entrypoint, Kong CLI setup, config loading
- `quad-ops/*.go` - Individual command implementations (one file per command)

## Commands

| Command | File | Description |
|---------|------|-------------|
| `sync` | sync.go | Sync repositories, generate Quadlet units, pull images, enable and start services (supports `--rollback`) |
| `validate` | validate.go | Validate compose files (by path or from configured repositories) |
| `update` | update.go | Self-update quad-ops binary |
| `version` | version.go | Print version info and check for updates |

## Conventions

### Command Implementation Pattern

Each command follows this structure:

```go
type FooCmd struct {
    // Command-specific flags defined here
}

func (f *FooCmd) Run(globals *Globals) error {
    // Access config via globals.AppCfg
    // Return error on failure, nil on success
}
```

Note: `UpdateCmd` and `VersionCmd` use `Run()` without `*Globals` since they don't need app config.

### Key Points

- Commands receive `*Globals` which contains `AppCfg *config.AppConfig`
- Always check `globals.AppCfg == nil` before using config
- Use `globals.AppCfg.GetRepositoryDir()` and `globals.AppCfg.GetQuadletDir()` for paths
- User mode is auto-detected via `config.IsUserMode()` (returns `true` if UID ≠ 0)
- Print progress with `fmt.Printf`, errors with `fmt.Printf("  ERROR: %v\n", err)`
- Track failures and return aggregate error (e.g., `"%d repository(ies) failed"`)
- Systemd client: use `systemd.New(ctx, systemd.ScopeAuto)` and always defer `Close()`

## Separation of Concerns

### What belongs in `cmd/`

Commands should only contain CLI-specific logic:

- **Argument parsing** - Kong struct tags and flag definitions
- **Output formatting** - Progress messages, warnings, error display
- **Orchestration** - Iterating over repositories/projects and calling internal packages
- **User interaction** - Prompts, confirmations, verbose output control
- **Exit code handling** - Aggregating errors and returning appropriate status

### What belongs in `internal/`

All business logic must live in internal packages:

| Package | Responsibility |
|---------|---------------|
| `config` | Configuration loading, path resolution, user/system mode detection |
| `git` | Repository cloning, fetching, checkout, commit hash retrieval |
| `compose` | Compose file loading, validation, secret checking, service filtering |
| `systemd` | Quadlet unit generation, systemd D-Bus operations (start/stop/reload) |
| `state` | Deployment state persistence (current/previous commit tracking for rollback) |

### Anti-patterns to avoid

- **Inline validation logic** - Move loops and conditionals that check business rules to internal packages
- **Direct external tool calls** - Wrap podman/systemd/git CLI invocations in internal packages for testability
- **Business rule duplication** - If multiple commands need the same check, add a function to an internal package
- **Complex data transformations** - Keep cmd functions thin; push logic to internal packages

### Dependencies

- `github.com/alecthomas/kong` - CLI framework (tags: `cmd:""`, `help:""`)
- `github.com/alecthomas/kong-yaml` - YAML config loader
- `github.com/creativeprojects/go-selfupdate` - Self-update mechanism (used by update/version)
- Internal packages: `config`, `git`, `compose`, `systemd`, `state`

## Testing

- Test files: `*_test.go` in same directory
- Use `t.TempDir()` for filesystem tests
- Test error conditions (nil config, empty repos)
- Helper functions should be package-private
