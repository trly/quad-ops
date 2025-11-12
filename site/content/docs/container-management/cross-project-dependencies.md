---
title: "Cross-Project Dependencies"
weight: 50
---

# Cross-Project Dependencies

Quad-Ops supports declaring dependencies on services in other projects using the `x-quad-ops-depends-on` extension field. This enables complex multi-project architectures where services can depend on infrastructure or platform services deployed separately.

## Overview

**Use Cases:**
- **Shared Infrastructure** - Application services depend on centrally-managed databases or message queues
- **Service Mesh** - Multiple projects route through a shared ingress proxy
- **Platform Services** - Applications depend on monitoring, logging, or observability infrastructure
- **Microservice Dependencies** - Services in different projects that communicate with each other

## Syntax

Add the `x-quad-ops-depends-on` extension field to any service:

```yaml
services:
  backend:
    image: myapp:latest
    x-quad-ops-depends-on:
      - project: infrastructure
        service: proxy
        optional: false  # Default: fail if not found
      - project: monitoring
        service: prometheus
        optional: true   # Warn if not found
```

### Fields

- **project** (required) - Name of the other project (must follow project naming rules)
- **service** (required) - Name of the service in that project (must follow service naming rules)
- **optional** (optional, default: false) - If true, warns when missing; if false, deployment fails

## How It Works

### 1. Validation

When you run `quad-ops up` or `quad-ops sync`, Quad-Ops:

1. Parses the `x-quad-ops-depends-on` field from your compose file
2. Validates that external services exist in the runtime (systemd or launchd)
3. **Required dependencies** - Fails deployment with clear error if missing
4. **Optional dependencies** - Logs warning if missing but continues deployment

### 2. Startup Ordering

External dependencies are included in the topological ordering graph:

```
infrastructure-proxy.service  (deployed in infrastructure project)
    ↓ (After + Requires)
app-backend.service          (this project)
```

Services start in dependency order, ensuring external dependencies are running before dependent services.

### 3. Platform Integration

**Linux (systemd/Quadlet):**
```ini
[Unit]
Description=app-backend container
After=infrastructure-proxy.service
Requires=infrastructure-proxy.service  # or Wants= for optional
```

**macOS (launchd):**
```xml
<key>DependsOn</key>
<array>
  <string>com.quad-ops.infrastructure.proxy</string>
</array>
```

## Complete Example

### Step 1: Deploy Infrastructure Project

```yaml
# ~/projects/infrastructure/compose.yml
name: infrastructure
services:
  proxy:
    image: nginx:latest
    ports:
      - "80:80"
  
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
```

```bash
cd ~/projects/infrastructure
quad-ops up
# Creates: infrastructure-proxy.service, infrastructure-db.service
```

### Step 2: Deploy Application Project with External Dependencies

```yaml
# ~/projects/myapp/compose.yml
name: myapp
services:
  backend:
    image: myapp:latest
    x-quad-ops-depends-on:
      - project: infrastructure
        service: proxy
      - project: infrastructure
        service: db
    depends_on:
      - redis  # Intra-project dependency
  
  redis:
    image: redis:latest
```

```bash
cd ~/projects/myapp
quad-ops up
# 1. Validates infrastructure-proxy and infrastructure-db exist
# 2. Generates myapp-backend.container with After=/Requires= for external deps
# 3. Starts in order: infrastructure-proxy, infrastructure-db → myapp-redis → myapp-backend
```

### Step 3: Updates Cascade Automatically

When infrastructure services restart, dependent services automatically restart (via systemd `Requires=`):

```bash
cd ~/projects/infrastructure
git pull
quad-ops sync
# infrastructure-proxy restarts → systemd auto-restarts myapp-backend
```

## Optional Dependencies

Use `optional: true` for graceful degradation when external services might not be available:

```yaml
services:
  app:
    image: myapp:latest
    x-quad-ops-depends-on:
      - project: monitoring
        service: prometheus
        optional: true  # Warn if missing, don't fail
```

**Behavior:**
- If `monitoring-prometheus` exists: Adds `After` + `Wants` (soft dependency)
- If `monitoring-prometheus` missing: Logs warning, continues deployment

**When to use optional dependencies:**
- Development/testing environments where monitoring isn't deployed
- Graceful degradation scenarios
- Feature flags that enable/disable integrations

## Naming Requirements

External project and service names must follow the same naming rules as regular names:

**Project names:** `^[a-z0-9][a-z0-9_-]*$`
- Must start with lowercase letter or digit
- Can contain: lowercase letters, digits, dashes, underscores

**Service names:** `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
- Must start with alphanumeric character
- Can contain: alphanumeric, dashes, underscores, periods

Invalid names will cause validation errors with clear messages.

## Limitations

### Same User/System Scope

External dependencies must be in the same scope (user vs system):

- ✅ User service → User service (both rootless)
- ✅ System service → System service (both root)
- ❌ User service → System service (cross-scope not supported by systemd)

### No Circular Dependencies

Circular dependencies across projects are not allowed:

```
project-a (service: api) → project-b (service: db)
                             ↓
project-b (service: cache) → project-a (service: auth)
```

This creates a cycle and will fail deployment. Design your architecture to avoid circular dependencies.

### Deployment Order Matters

External services must be deployed before dependent projects:

1. Deploy `infrastructure` project first (`quad-ops up`)
2. Then deploy `app` project that depends on it

If you try to deploy `app` first, validation will fail with:
```
Error: required external service not found: infrastructure-proxy
Ensure projects are deployed in correct order
```

## Troubleshooting

### Error: "required external service not found"

**Cause:** The external service hasn't been deployed yet.

**Fix:** Deploy the dependency project first:
```bash
cd ~/projects/infrastructure
quad-ops up

cd ~/projects/myapp
quad-ops up  # Now succeeds
```

### Error: "invalid project name" in x-quad-ops-depends-on

**Cause:** Project name doesn't follow naming requirements.

**Fix:** Update to use valid project name matching `^[a-z0-9][a-z0-9_-]*$`:
```yaml
# ❌ Invalid
x-quad-ops-depends-on:
  - project: My-Infrastructure  # Uppercase
    service: proxy

# ✅ Valid
x-quad-ops-depends-on:
  - project: infrastructure
    service: proxy
```

### Warning: "optional external dependency not found"

**Cause:** Optional dependency doesn't exist (this is not an error).

**Action:** 
- If the service should exist, deploy it
- If it's expected to be missing in this environment, no action needed

## Best Practices

1. **Deploy dependencies first** - Always deploy infrastructure/platform projects before dependent applications
2. **Use optional wisely** - Only mark dependencies as optional if your app can truly function without them
3. **Document dependencies** - Add comments in compose files explaining why external dependencies exist
4. **Consistent naming** - Use consistent project naming across your infrastructure
5. **Separate concerns** - Group related infrastructure services in dedicated projects

## See Also

- [Docker Compose Support](../docker-compose-support) - Full list of supported Compose features
- [Repository Structure](../repository-structure) - Organizing multiple projects
- [Naming Requirements](#naming-requirements) - Project and service naming rules
