---
title: "Environment Files"
weight: 30
---

# Environment Files

Quad-Ops automatically discovers and processes environment files to provide configuration flexibility and secure handling of sensitive data.

## Environment File Discovery

### Standard Environment File

The `.env` file in the same directory as the Docker Compose file is automatically loaded:

```
project/
├── docker-compose.yml
└── .env                 # ✅ Automatically loaded
```

### Service-Specific Environment Files

For each service, Quad-Ops searches for service-specific environment files in this order:

1. `.env.<service-name>` - Hidden file with service suffix
2. `<service-name>.env` - Service name with .env extension
3. `env/<service-name>.env` - In env subdirectory
4. `envs/<service-name>.env` - In envs subdirectory

**Example for service named `webapp`:**
```
project/
├── docker-compose.yml
├── .env                 # Global environment
├── .env.webapp          # ✅ Service-specific (highest priority)
├── webapp.env           # ✅ Alternative naming
├── env/
│   └── webapp.env       # ✅ In env directory
└── envs/
    └── webapp.env       # ✅ In envs directory
```

## Environment File Processing

### Variable Interpolation in Compose Files

Environment variables are substituted in Docker Compose files during processing:

**`.env` file:**
```env
APP_VERSION=1.0.0
DB_NAME=myapp
PORT=8080
```

**`docker-compose.yml`:**
```yaml
version: '3.8'
services:
  web:
    image: myapp:${APP_VERSION}
    ports:
      - "${PORT}:80"
    environment:
      - DATABASE_NAME=${DB_NAME}
```

**Processed result:**
```yaml
services:
  web:
    image: myapp:1.0.0
    ports:
      - "8080:80"
    environment:
      - DATABASE_NAME=myapp
```

### Service Environment Configuration

Environment files are added to Quadlet units using the `EnvironmentFile` directive:

**Service-specific file:** `.env.webapp`
```env
NODE_ENV=production
API_URL=https://api.example.com
SESSION_SECRET=super-secret-key
```

**Generated Quadlet unit:**
```ini
[Container]
Image=myapp:latest
EnvironmentFile=.env.webapp
NetworkAlias=webapp
```

## Environment File Examples

### Development Configuration

**`.env`:**
```env
# Global development settings
NODE_ENV=development
DEBUG=true
LOG_LEVEL=debug
```

**`.env.api`:**
```env
# API service specific
API_PORT=3000
DATABASE_URL=postgresql://localhost:5432/dev_db
REDIS_URL=redis://localhost:6379
```

**`.env.frontend`:**
```env
# Frontend service specific
REACT_APP_API_URL=http://localhost:3000
REACT_APP_DEBUG=true
```

### Production Configuration

**`.env`:**
```env
# Global production settings
NODE_ENV=production
DEBUG=false
LOG_LEVEL=info
```

**`.env.api`:**
```env
# API service specific
API_PORT=8080
DATABASE_URL=postgresql://db:5432/prod_db
REDIS_URL=redis://cache:6379
JWT_SECRET=prod-jwt-secret-key
```

**`.env.frontend`:**
```env
# Frontend service specific
REACT_APP_API_URL=https://api.example.com
REACT_APP_DEBUG=false
```

### Multi-Environment Structure

```
project/
├── docker-compose.yml
├── environments/
│   ├── dev/
│   │   ├── .env
│   │   ├── .env.api
│   │   └── .env.frontend
│   ├── staging/
│   │   ├── .env
│   │   ├── .env.api
│   │   └── .env.frontend
│   └── prod/
│       ├── .env
│       ├── .env.api
│       └── .env.frontend
```

## Environment Variable Syntax

### Basic Variables

```env
# Simple key-value pairs
DATABASE_HOST=localhost
DATABASE_PORT=5432
DEBUG=true
```

### Quoted Values

```env
# Handle spaces and special characters
APP_NAME="My Application"
DATABASE_URL="postgresql://user:pass@host:5432/db"
SECRET_KEY='complex!@#$%^&*()_+secret'
```

### Multi-line Values

```env
# Multi-line values (use quotes)
SSL_CERT="-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END CERTIFICATE-----"
```

### Comments

```env
# Database configuration
DATABASE_HOST=localhost  # Local development database
DATABASE_PORT=5432      # Default PostgreSQL port

# API configuration
API_VERSION=v1          # Current API version
```

## Security Best Practices

### Sensitive Data Handling

**❌ Avoid committing secrets to Git:**
```env
# .env (DO NOT COMMIT)
DATABASE_PASSWORD=secret123
API_KEY=sk_live_abcdef123456
JWT_SECRET=super-secret-key
```

**✅ Use environment-specific files:**
```bash
# .gitignore
.env.local
.env.*.local
*.secret
secrets/
```

**✅ Template files for documentation:**
```env
# .env.example (safe to commit)
DATABASE_PASSWORD=your_secure_password_here
API_KEY=your_api_key_here
JWT_SECRET=generate_a_secure_jwt_secret
```

### File Permissions

Protect sensitive environment files:

```bash
# Restrict access to environment files
chmod 600 .env*
chmod 600 env/*.env
chmod 600 envs/*.env

# Verify permissions
ls -la .env*
```

### Service-Specific Isolation

Use service-specific files to limit secret exposure:

```
project/
├── .env                 # Non-sensitive global vars
├── .env.api            # API secrets (DB, external APIs)
├── .env.frontend       # Frontend config (public API URLs)
└── .env.worker         # Worker-specific config
```

## Environment Variable Precedence

Variables are resolved in this order (highest to lowest priority):

1. **Container environment** (from `environment:` in compose)
2. **Service-specific env files** (`.env.service`)
3. **Global env file** (`.env`)
4. **System environment** (from shell)

### Example Precedence

**`.env`:**
```env
APP_ENV=development
DATABASE_HOST=localhost
API_PORT=3000
```

**`.env.api`:**
```env
APP_ENV=staging        # Overrides global setting
DATABASE_HOST=db       # Overrides global setting
# API_PORT inherited from .env (3000)
```

**`docker-compose.yml`:**
```yaml
services:
  api:
    environment:
      - APP_ENV=production  # Highest priority - overrides both files
      # DATABASE_HOST=db from .env.api
      # API_PORT=3000 from .env
```

**Final environment for api service:**
```env
APP_ENV=production      # From compose environment
DATABASE_HOST=db        # From .env.api
API_PORT=3000          # From .env
```

## Special Characters and Escaping

### Handling Special Characters

Environment files with special characters benefit from service-specific files:

**Problematic in compose interpolation:**
```env
# .env - Can cause issues with compose parsing
PASSWORD=p@ssw*rd!2023
COMMAND=echo "hello world"
```

**Safe in service-specific files:**
```env
# .env.api - Passed directly to container
PASSWORD=p@ssw*rd!2023
COMMAND=echo "hello world"
DATABASE_URL=postgresql://user:p@ss@host/db?param=value&other=123
```

### Escaping Guidelines

For compose file interpolation:
```env
# Use quotes for complex values
DATABASE_URL="postgresql://user:p@ss@host/db"
SHELL_COMMAND="echo 'Hello World'"

# Escape special characters
REGEX_PATTERN="^[a-zA-Z0-9_]+\$$"
```

### Debugging Environment Variables

**Check variable resolution:**
```bash
# Test compose file processing
docker-compose -f docker-compose.yml config

# Check what variables are available
cat .env .env.* 2>/dev/null
```

**Verify generated units:**
```bash
# Check EnvironmentFile directives
grep -r "EnvironmentFile" /etc/containers/systemd/

# View specific unit
cat /etc/containers/systemd/myapp-service.container
```

**Test container environment:**
```bash
# Check running container environment
podman exec myapp-service env

# View specific variable
podman exec myapp-service echo "$DATABASE_URL"
```

### Environment File Validation

**Check file syntax:**
```bash
# Verify no syntax errors
source .env && echo "Syntax OK"

# Check for common issues
grep -n "=" .env | grep -v "^[A-Z_][A-Z0-9_]*="
```

**Validate variable names:**
```bash
# Check for invalid variable names
grep -E "^[^A-Z_]" .env
grep -E "^[0-9]" .env
```

## Integration Examples

### GitOps Workflow

**Development repository:**
```
my-app/
├── docker-compose.yml
├── .env.example        # Template (committed)
├── .env               # Local dev (ignored)
└── .env.api           # API config (ignored)
```

**Production repository:**
```
my-app-deploy/
├── docker-compose.yml  # Same as dev
├── environments/
│   ├── staging/
│   │   ├── .env
│   │   └── .env.api
│   └── production/
│       ├── .env
│       └── .env.api
```

### CI/CD Integration

```bash
#!/bin/bash
# deployment script

ENV_NAME=${1:-staging}

# Copy environment files
cp environments/$ENV_NAME/.env .
cp environments/$ENV_NAME/.env.* .

# Deploy with quad-ops
quad-ops sync --repository myapp
```

## Next Steps

- [Build Support](build-support) - Docker build configurations
- [Repository Structure](repository-structure) - Understanding file discovery
- [Docker Compose Support](docker-compose-support) - Complete compose reference