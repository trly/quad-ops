# Cross-Project Dependencies Example

This example demonstrates Quad-Ops' cross-project dependency feature using the `x-quad-ops-depends-on` extension.

## Architecture

```
infrastructure/          (deployed first)
  ├── proxy (nginx)
  └── db (postgres)
       ↑ ↑
       │ └──────────────┐
       │                │
app/                    │ (deployed second, depends on infrastructure)
  ├── backend ──────────┤
  │    └── redis (intra-project dependency)
  └── redis
```

## Projects

### 1. Infrastructure Project

Provides shared services used by multiple applications:
- **proxy** - Nginx reverse proxy on port 80
- **db** - PostgreSQL database

### 2. App Project

Application services that depend on infrastructure:
- **backend** - Node.js application
  - Depends on `infrastructure/proxy` (cross-project)
  - Depends on `infrastructure/db` (cross-project)
  - Depends on `redis` (intra-project)
- **redis** - Redis cache

## Deployment Steps

### Step 1: Deploy Infrastructure

```bash
cd infrastructure
quad-ops up
```

Verify services are running:
```bash
systemctl --user status infrastructure-proxy.service
systemctl --user status infrastructure-db.service
```

### Step 2: Deploy Application

```bash
cd ../app
quad-ops up
```

This will:
1. Validate that `infrastructure-proxy` exists ✓
2. Validate that `infrastructure-db` exists ✓
3. Generate systemd units with proper dependencies
4. Start services in order: redis → backend (after infrastructure services)

### Step 3: Verify Dependencies

Check that systemd dependencies are correct:

```bash
systemctl --user list-dependencies app-backend.service
```

Should show:
```
app-backend.service
├─infrastructure-proxy.service
├─infrastructure-db.service
└─app-redis.service
```

## Key Features Demonstrated

### 1. Cross-Project Dependencies

```yaml
x-quad-ops-depends-on:
  - project: infrastructure
    service: proxy
    optional: false  # Fail if not found
```

### 2. Validation

If infrastructure isn't deployed, `quad-ops up` fails:
```
Error: required external service not found: infrastructure-proxy
Ensure projects are deployed in correct order
```

### 3. Startup Ordering

Services start in dependency order:
1. `infrastructure-proxy` (external)
2. `infrastructure-db` (external)
3. `app-redis` (local dependency)
4. `app-backend` (depends on all above)

### 4. Automatic Restarts

When infrastructure services restart, dependent services restart automatically:

```bash
cd infrastructure
git pull
quad-ops sync
# infrastructure-proxy restarts → systemd auto-restarts app-backend
```

## Testing Optional Dependencies

Modify `app/compose.yml` to make dependencies optional:

```yaml
x-quad-ops-depends-on:
  - project: infrastructure
    service: proxy
    optional: true  # Warn if missing, don't fail
```

Now you can deploy app even if infrastructure isn't ready (logs warning instead of failing).

## Cleanup

```bash
# Stop and remove app services
cd app
quad-ops down

# Stop and remove infrastructure services
cd ../infrastructure
quad-ops down
```

## See Also

- [Cross-Project Dependencies Documentation](https://trly.github.io/quad-ops/docs/container-management/cross-project-dependencies/)
- [Naming Requirements](https://trly.github.io/quad-ops/docs/container-management/naming-requirements/)
