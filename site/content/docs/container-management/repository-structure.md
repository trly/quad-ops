---
title: "Repository Structure"
weight: 10
---

# Repository Structure

This guide details how Quad-Ops discovers, reads, and processes files from Git repositories.

## File Discovery Process

Quad-Ops follows a systematic approach to process repository contents:

1. **Repository Clone/Update** - Git operations to get latest content
2. **Directory Scanning** - Looking for Docker Compose files
3. **File Validation** - Ensuring files are valid Docker Compose format
4. **Environment Discovery** - Finding associated environment files
5. **Content Processing** - Converting to Quadlet units

## Docker Compose File Discovery

### Supported File Names

Quad-Ops automatically detects Docker Compose files with these names:

- `docker-compose.yml`
- `docker-compose.yaml`
- `compose.yml`
- `compose.yaml`

### Search Locations

#### Root Directory Search
When `composeDir` is not specified:

```
repository/
├── docker-compose.yml     ✅ Found
├── src/
├── docs/
└── README.md
```

#### Subdirectory Search
When `composeDir` is specified:

```yaml
repositories:
  - name: myapp
    url: https://github.com/user/app.git
    composeDir: deploy
```

```
repository/
├── src/
├── deploy/
│   ├── docker-compose.yml  ✅ Found (in deploy/)
│   └── .env
├── docs/
└── README.md
```

#### Multiple Compose Files
If multiple compose files exist, they are processed in this priority order:

1. `docker-compose.yml`
2. `docker-compose.yaml`
3. `compose.yml`
4. `compose.yaml`

Only the first found file is processed per directory.

## Project Naming Convention

Quad-Ops generates project names using this format:
```
<repository-name>-<directory-path>
```

### Examples

| Repository | composeDir | Compose Location | Project Name |
|------------|------------|------------------|--------------|
| `myapp` | *(empty)* | `docker-compose.yml` | `myapp` |
| `services` | `api` | `api/docker-compose.yml` | `services-api` |
| `infra` | `database/postgres` | `database/postgres/compose.yml` | `infra-database-postgres` |

### Multi-Directory Support

A single repository can contain multiple Docker Compose projects:

```
repository/
├── frontend/
│   └── docker-compose.yml    # Project: myrepo-frontend
├── backend/
│   └── docker-compose.yml    # Project: myrepo-backend
└── database/
    └── compose.yml           # Project: myrepo-database
```

Each directory with a compose file becomes a separate project.

## Directory Structure Examples

### Simple Application

```
my-app/
├── docker-compose.yml
├── .env
├── app/
│   └── Dockerfile
└── nginx/
    └── nginx.conf
```

**Configuration:**
```yaml
repositories:
  - name: my-app
    url: https://github.com/user/my-app.git
```

**Result:** Project name `my-app`

### Microservices Repository

```
microservices/
├── auth-service/
│   ├── docker-compose.yml
│   ├── .env.auth
│   └── Dockerfile
├── user-service/
│   ├── docker-compose.yml
│   ├── .env.user
│   └── Dockerfile
└── api-gateway/
    ├── compose.yml
    └── .env
```

**Configuration:**
```yaml
repositories:
  - name: microservices
    url: https://github.com/company/microservices.git
```

**Result:** Three projects:
- `microservices-auth-service`
- `microservices-user-service`
- `microservices-api-gateway`

### Environment-Based Structure

```
webapp/
├── environments/
│   ├── dev/
│   │   ├── docker-compose.yml
│   │   └── .env
│   ├── staging/
│   │   ├── docker-compose.yml
│   │   └── .env
│   └── prod/
│       ├── docker-compose.yml
│       └── .env
└── src/
```

**Configuration:**
```yaml
repositories:
  - name: webapp-dev
    url: https://github.com/company/webapp.git
    composeDir: environments/dev

  - name: webapp-prod
    url: https://github.com/company/webapp.git
    composeDir: environments/prod
```

**Result:** Two projects:
- `webapp-dev-environments-dev`
- `webapp-prod-environments-prod`

## File Processing Order

### 1. Git Operations
```bash
# First sync or clone
git clone <repository-url> <local-path>

# Subsequent syncs
git fetch origin
git reset --hard origin/<ref>
```

### 2. Directory Scanning
```bash
# Find all directories containing compose files
find <repository> -name "docker-compose.y*ml" -o -name "compose.y*ml"
```

### 3. File Validation
- Parse YAML syntax
- Validate Docker Compose schema
- Check for required sections (services)

### 4. Environment File Discovery
For each compose file directory, scan for:
- `.env` (standard environment file)
- `.env.<service>` (service-specific files)
- `<service>.env`
- `env/<service>.env`
- `envs/<service>.env`

### 5. Content Processing
- Variable interpolation using environment files
- Service, volume, and network extraction
- Quadlet unit generation
- Dependency resolution

## Working Directory Behavior

Quad-Ops processes each compose file in its containing directory, ensuring:
- **Relative paths** work correctly for bind mounts
- **Build contexts** resolve properly for Dockerfile builds
- **Environment files** are found in the expected locations

### Example Processing

For this structure:
```
repository/
└── apps/
    ├── web/
    │   ├── docker-compose.yml
    │   ├── .env
    │   └── html/
    └── api/
        ├── compose.yml
        └── .env.api
```

Processing occurs as:
1. `cd repository/apps/web && process docker-compose.yml`
2. `cd repository/apps/api && process compose.yml`

This ensures relative paths like `./html:/var/www/html` work correctly.

## Best Practices

### Repository Organization
1. **Keep compose files close to related code**
2. **Use consistent naming** across repositories
3. **Group related services** in subdirectories
4. **Include environment files** in the same directory

### File Naming
1. **Use standard names** (`docker-compose.yml`, `compose.yml`)
2. **Avoid special characters** in directory names
3. **Keep paths reasonable length** for systemd unit names

### Directory Structure
1. **Separate environments** into different directories or repositories
2. **Group microservices** logically
3. **Include documentation** alongside compose files
4. **Use `.gitignore`** for local-only files

### Debugging Commands

```bash
# List discovered projects
quad-ops unit list -t all

# Check repository sync status
quad-ops sync --verbose

# Validate specific compose file
docker-compose -f path/to/compose.yml config
```

## Next Steps

- [Docker Compose Support](docker-compose-support) - Learn about supported compose features
- [Environment Files](environment-files) - Understand environment file processing