# quad-ops Agent Guidelines

GitOps framework for Podman containers on Linux and macOS.

## Commands

- **Build**: `task build` (fmt, lint, test, compile)
- **Test all**: `task test` or `gotestsum --format pkgname --format-icons text -- -coverprofile=coverage.out -v ./...`
- **Test single**: `go test -run TestName ./path/to/package -v`
- **Lint**: `task lint` or `golangci-lint run`
- **Format**: `task fmt` or `go fmt ./...`

## Architecture

Core pipeline: `Docker Compose → Spec Converter → Platform Renderer → Lifecycle Manager`.

### Key Packages

- `internal/compose/` - Parses Docker Compose files, converts to platform-agnostic service specs
- `internal/platform/systemd/` - Quadlet unit file renderer for Linux
- `internal/platform/launchd/` - Plist renderer for macOS
- `internal/service/` - Core service specification models (models.go, validate.go)
- `internal/repository/` - Unit file storage and git synchronization
- `internal/systemd/` - systemd/DBus integration for lifecycle management
- `cmd/` - CLI commands using Cobra (sync, up, down, daemon, etc.)

## Code Style

- **Testing**: table-driven tests preferred, heavy use of dependency injection and mocks
- **Imports**: Group stdlib, external packages, then internal (`github.com/trly/quad-ops/internal/*`)
- **Comments**: Package-level godoc required, exported functions documented
- **Error handling**: Wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Naming**: Service specs use sanitized names via `service.SanitizeName()`, unit files prefixed with project name
- **Validation**: All specs validated via `spec.Validate()` before rendering
- **Linters**: errcheck, govet, staticcheck, unused, revive, gosec, misspell enabled via golangci-lint

## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**
```bash
bd ready --json
```

**Create new issues:**
```bash
bd create "Issue title" -t bug|feature|task -p 0-4 --json
bd create "Issue title" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**
```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**
```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`
6. **Commit together**: Always commit the `.beads/issues.jsonl` file together with the code changes so issue state stays in sync with code state

### Auto-Sync

bd automatically syncs with git:
- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### MCP Server (Recommended)

If using Claude or MCP-compatible clients, install the beads MCP server:

```bash
pip install beads-mcp
```

Add to MCP config (e.g., `~/.config/claude/config.json`):
```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

Then use `mcp__beads__*` functions instead of CLI commands.

### Managing AI-Generated Planning Documents

AI assistants often create planning and design documents during development:
- PLAN.md, IMPLEMENTATION.md, ARCHITECTURE.md
- DESIGN.md, CODEBASE_SUMMARY.md, INTEGRATION_PLAN.md
- TESTING_GUIDE.md, TECHNICAL_DESIGN.md, and similar files

**Best Practice: Use a dedicated directory for these ephemeral files**

**Recommended approach:**
- Create a `history/` directory in the project root
- Store ALL AI-generated planning/design docs in `history/`
- Keep the repository root clean and focused on permanent project files
- Only access `history/` when explicitly asked to review past planning

**Example .gitignore entry (optional):**
```
# AI planning documents (ephemeral)
history/
```

**Benefits:**
- ✅ Clean repository root
- ✅ Clear separation between ephemeral and permanent documentation
- ✅ Easy to exclude from version control if desired
- ✅ Preserves planning history for archeological research
- ✅ Reduces noise when browsing the project

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `history/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md and QUICKSTART.md.

## Change Validation

Always Build after a set of changes are completed to ensure tests pass and the application has no compilation issues.
