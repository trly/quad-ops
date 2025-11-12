---
title: "Naming Requirements"
weight: 10
---

# Naming Requirements

Quad-Ops enforces strict naming requirements for project and service names following the Docker Compose specification. Invalid names are rejected with clear error messages.

## Why Strict Validation?

Quad-Ops validates names strictly instead of silently transforming them:

**Benefits:**
- **Predictable** - What you write is what the system uses
- **Debuggable** - No hidden transformations
- **Secure** - No edge cases where invalid data gets through
- **Spec-compliant** - Follows Docker Compose specification exactly

## Project Names

Project names identify your compose project and are used as prefixes for all generated service units.

### Pattern

`^[a-z0-9][a-z0-9_-]*$`

### Requirements

- Must start with lowercase letter or digit
- Can contain only: lowercase letters, digits, dashes (`-`), underscores (`_`)
- Cannot be empty
- Cannot start with dash or underscore

### Valid Examples

```yaml
name: myproject       # ✅ Simple name
name: my-project      # ✅ With dashes
name: my_project      # ✅ With underscores
name: project123      # ✅ Ends with numbers
name: 123project      # ✅ Starts with number
name: my-project-v2   # ✅ Complex name
```

### Invalid Examples

```yaml
name: My-Project      # ❌ Uppercase letters
name: my_project!     # ❌ Special character (!)
name: _myproject      # ❌ Starts with underscore
name: -myproject      # ❌ Starts with dash
name: my project      # ❌ Contains space
name: my.project      # ❌ Period not allowed in project names
name: ""              # ❌ Empty string
```

### Error Message

```
Error: invalid project name "My-Project": must contain only lowercase 
letters, digits, dashes, underscores and start with letter/digit
```

## Service Names

Service names identify individual containers within your project.

### Pattern

`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`

### Requirements

- Must start with alphanumeric character (uppercase or lowercase)
- Can contain: alphanumeric characters, dashes (`-`), underscores (`_`), periods (`.`)
- Cannot be empty
- Cannot start with dash, underscore, or period

### Valid Examples

```yaml
services:
  web:             # ✅ Simple lowercase
    image: nginx
  
  Web:             # ✅ Uppercase allowed
    image: nginx
  
  web-api:         # ✅ With dashes
    image: nginx
  
  web_api:         # ✅ With underscores
    image: nginx
  
  web.api:         # ✅ With periods
    image: nginx
  
  api-v2:          # ✅ Complex name
    image: nginx
  
  Service123:      # ✅ Mixed case with numbers
    image: nginx
```

### Invalid Examples

```yaml
services:
  _web:            # ❌ Starts with underscore
    image: nginx
  
  -web:            # ❌ Starts with dash
    image: nginx
  
  .web:            # ❌ Starts with period
    image: nginx
  
  web api:         # ❌ Contains space
    image: nginx
  
  web@api:         # ❌ Special character (@)
    image: nginx
```

### Error Message

```
Error: invalid service name "_web": must contain only alphanumeric 
characters, underscores, periods, dashes and start with alphanumeric
```

## How Project Names Are Determined

Quad-Ops determines the project name from one of two sources:

### 1. Explicit `name:` Field (Recommended)

```yaml
name: myproject  # ✅ Explicit project name
services:
  web:
    image: nginx
```

### 2. Directory Name (Fallback)

If no `name:` field is specified, the directory containing `compose.yml` is used:

```bash
/home/user/myproject/compose.yml  # Project name: "myproject"
/home/user/my-app/compose.yml     # Project name: "my-app"
```

**Important:** The directory name must also follow project naming requirements!

## Fixing Invalid Names

If you have existing compose files with invalid names, follow these steps:

1. **Validate existing compose files:**
   ```bash
   quad-ops validate /path/to/compose.yml
   ```

2. **Fix any naming errors:**
   ```yaml
   # Before
   name: My-Project
   services:
     Web-App:
       image: nginx
   
   # After
   name: my-project
   services:
     web-app:
       image: nginx
   ```

3. **Re-validate:**
   ```bash
   quad-ops validate /path/to/compose.yml
   # Should pass without errors
   ```

## Common Name Fixes

| Invalid Name | Common Fix |
|--------------|------------|
| `My-Project` | `my-project` |
| `my_project!` | `my-project` |
| `_myproject` | `myproject` |
| `-my-project` | `my-project` |
| `My Service` | `my-service` |

## Validation Command

Check your compose files for naming errors:

```bash
# Validate single file
quad-ops validate compose.yml

# Validate directory
quad-ops validate /path/to/compose-files/

# Validate with verbose output
quad-ops validate --verbose compose.yml
```

The validation command checks:
- Project name validity
- Service name validity
- Compose syntax and structure
- Quad-ops extension compatibility

## Directory-Based Project Names

When using directory names as project names, ensure the directory itself has a valid name:

```bash
# ❌ Invalid directory names
/home/user/My-Project/compose.yml       # Uppercase
/home/user/_my_project/compose.yml      # Starts with underscore

# ✅ Valid directory names
/home/user/my-project/compose.yml       # Lowercase with dashes
/home/user/myproject/compose.yml        # Simple lowercase
```

**Workaround:** Use explicit `name:` field in compose file:

```yaml
# In /home/user/My-Project/compose.yml
name: my-project  # ✅ Overrides invalid directory name
services:
  web:
    image: nginx
```

## Impact on Generated Units

Project and service names affect generated systemd/launchd unit names:

**Project name:** `my-project`  
**Service name:** `web`

**Generated systemd unit:** `my-project-web.service`  
**Generated launchd label:** `com.quad-ops.my-project.web`

This is why naming rules are strict - they must be compatible with systemd/launchd naming requirements.

## See Also

- [Docker Compose Support](../docker-compose-support) - Full list of supported Compose features
- [Cross-Project Dependencies](../cross-project-dependencies) - Using valid names in external dependencies
