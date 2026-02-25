# quad-ops Site Agent Guidelines

## Commands

- **Dev server**: `hugo server -D` (serves locally with drafts)
- **Dev server (background)**: `tmux new-session -d -s hugo-dev 'cd site && hugo server -D'` (from project root)
- **Build**: `hugo` (builds static site to public/ directory)
- **Build with drafts**: `hugo -D`
- **Clean**: `rm -rf public/`

## Architecture

Hugo static site using the [hugo-book](https://github.com/alex-shpak/hugo-book) theme for documentation.

### Directory Structure

- `content/` - Markdown content files and documentation pages
- `static/` - Static assets (images, CSS, JS) served at site root
- `public/` - Generated static site (git-ignored, created by `hugo` build)
- `archetypes/` - Content templates for `hugo new` command
- `resources/` - Hugo resource cache (git-ignored)
- `hugo.toml` - Site configuration and theme settings

### Site Configuration

- **Base URL**: https://trly.github.io/quad-ops
- **Theme**: hugo-book with auto dark/light theme
- **Logo**: `static/images/quad-ops.svg`
- **Repository**: GitHub integration for "Edit this page" links
- **Content**: Markdown files in `content/` directory

## Content Guidelines

### File Organization
- Use meaningful directory structure under `content/`
- Index pages should be named `_index.md`
- Regular pages use `pagename.md`
- Follow hugo-book theme conventions for navigation

### Markdown Features
- Front matter in YAML format with `title`, `weight`, and other metadata
- Hugo shortcodes available for enhanced functionality
- Emoji support enabled (`:emoji_name:`)
- Cross-references using Hugo's `ref` and `relref` shortcodes

### Development Workflow
1. Start dev server with `hugo server -D`
2. Edit content in `content/` directory
3. Preview changes at http://localhost:1313
4. Build final site with `hugo` for deployment

## Deployment

Site deploys to GitHub Pages at https://trly.github.io/quad-ops from the `public/` directory contents.
