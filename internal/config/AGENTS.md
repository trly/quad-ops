# internal/config

Provides application configuration structures and path resolution utilities.

## Design

- `AppConfig` is deserialized from YAML configuration files
- Path defaults are determined by user vs system mode (`IsUserMode` checks uid)
- User mode paths use XDG-style directories (`~/.local/share`, `~/.local/state`, `~/.config`)
- System mode paths use standard system directories (`/var/lib`, `/etc`)

## Conventions

- Repository definitions are inline structs within `AppConfig` — keep them there unless complexity demands extraction
- Path-returning methods must respect the configured override first, then fall back to mode-based defaults
- Use `yaml` struct tags for all config fields
