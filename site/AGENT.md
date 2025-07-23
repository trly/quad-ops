# Agent Guidelines for quad-ops Documentation Site

## Overview
The documentation site for quad-ops is built using Hugo Extended and hosted at https://trly.github.io/quad-ops/

## Architecture & Structure
- **site/**: Hugo documentation source directory  
- **content/**: Markdown content files
- **themes/**: Hugo theme files
- **static/**: Static assets (images, CSS, JS)
- **config/**: Hugo configuration files

## Key Commands

### Development
- `hugo server` - Start local development server
- `hugo server -D` - Include draft content in development
- `hugo` - Build static site for production
- `hugo new content/page.md` - Create new content page

### Content Management
- All content written in Markdown format
- Front matter in YAML format for metadata
- Draft content excluded from production builds
- Automatic deployment via GitHub Actions

## Dependencies & Tools
- **hugo-extended**: Static site generator with SCSS support
- **GitHub Actions**: Automated deployment pipeline
- **GitHub Pages**: Hosting platform

## Deployment
- Source in `site/` directory
- GitHub Actions automated deployment to GitHub Pages
- Production site: https://trly.github.io/quad-ops/
- Automatic builds on push to main branch
- Managed with [docs.yaml](../.github/workflows/docs.yaml)

## Content Guidelines
- Follow Hugo documentation best practices
- Use clear, concise language for technical documentation
- Include code examples and configuration samples
- Maintain consistent formatting and structure
