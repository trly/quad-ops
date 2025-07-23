# Agent Guidelines for validate Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `validate` package provides security validation functions for sensitive data and system requirements checking. It includes input validation, secret security, environment variable validation, and system tool verification.

## Key Structures and Interfaces

### Core Structures
- **`SecretValidator`** - Validates secrets and sensitive data:
  - `allowedChars` - Regex for allowed characters
  - Methods for validating secret names, values, targets, and environment variables

- **`CommandRunner`** - Interface for executing system commands:
  - `RealCommandRunner` - Production implementation
  - Mock implementations for testing

### Core Functions
- **`SystemRequirements()`** - Validates system tool availability (systemd, podman)
- **`NewSecretValidator()`** - Creates a new secret validator instance
- **`EnvKey(key string)`** - Validates environment variable keys
- **`SanitizeForLogging(key, value string)`** - Sanitizes sensitive data for logging

### Security Constants
- `MaxSecretFileSize` - 1MB maximum for secret files
- `MaxEnvValueSize` - 32KB maximum for environment variable values
- `MaxSecretNameLen` - 253 characters (DNS name limit)
- `MaxEnvKeyLen` - 256 characters maximum for env keys
- `MaxSecretTargetLen` - 4KB maximum for secret target paths

## Usage Patterns

### System Requirements Validation
```go
// Check if required system tools are available
if err := validate.SystemRequirements(); err != nil {
    return fmt.Errorf("system requirements not met: %w", err)
}
```

### Secret Validation
```go
// Create validator and validate secrets
validator := validate.NewSecretValidator()

// Validate secret name
if err := validator.ValidateSecretName(secretName); err != nil {
    return fmt.Errorf("invalid secret name: %w", err)
}

// Validate secret value
if err := validator.ValidateSecretValue(secretContent); err != nil {
    return fmt.Errorf("invalid secret value: %w", err)
}

// Validate target path
if err := validator.ValidateSecretTarget(targetPath); err != nil {
    return fmt.Errorf("invalid secret target: %w", err)
}
```

### Environment Variable Validation
```go
// Validate environment variable key
if err := validate.EnvKey(key); err != nil {
    return fmt.Errorf("invalid env key: %w", err)
}

// Validate environment variable value
validator := validate.NewSecretValidator()
if err := validator.ValidateEnvValue(key, value); err != nil {
    return fmt.Errorf("invalid env value: %w", err)
}
```

### Safe Logging
```go
// Sanitize potentially sensitive values for logging
safeValue := validate.SanitizeForLogging(key, value)
log.GetLogger().Debug("Environment variable set", "key", key, "value", safeValue)
```

## Development Guidelines

### Security-First Approach
- All validation functions assume untrusted input
- Size limits prevent resource exhaustion attacks
- Pattern matching prevents injection attacks
- Sensitive data detection and sanitization

### Validation Strategy
- **Allowlist Approach**: Only allow known-safe patterns
- **Size Limits**: Prevent excessive resource usage
- **Content Inspection**: Check for dangerous patterns
- **Path Safety**: Prevent directory traversal

### Error Handling
- Clear, descriptive error messages
- No sensitive data in error messages
- Consistent error format across validators
- Security context preserved in errors

## Common Patterns

### Comprehensive Secret Validation
```go
func validateSecret(secret types.ServiceConfigObjSecret) error {
    validator := validate.NewSecretValidator()
    
    // Validate secret name
    if err := validator.ValidateSecretName(secret.Source); err != nil {
        return fmt.Errorf("invalid secret name %s: %w", secret.Source, err)
    }
    
    // Validate target path if specified
    if secret.Target != "" {
        if err := validator.ValidateSecretTarget(secret.Target); err != nil {
            return fmt.Errorf("invalid secret target %s: %w", secret.Target, err)
        }
    }
    
    return nil
}
```

### Safe Environment Processing
```go
func processEnvironmentVariables(envVars map[string]*string) error {
    validator := validate.NewSecretValidator()
    
    for key, value := range envVars {
        // Validate key format
        if err := validate.EnvKey(key); err != nil {
            log.GetLogger().Warn("Invalid environment variable key", "key", key, "error", err)
            continue
        }
        
        // Validate value if present
        if value != nil {
            if err := validator.ValidateEnvValue(key, *value); err != nil {
                log.GetLogger().Warn("Invalid environment variable value", "key", key, "error", err)
                continue
            }
        }
        
        // Set with safe logging
        safeValue := validate.SanitizeForLogging(key, *value)
        log.GetLogger().Debug("Processing environment variable", "key", key, "value", safeValue)
    }
    
    return nil
}
```

## Validation Rules

### Secret Names
- Must follow DNS naming conventions
- Maximum 253 characters (DNS limit)
- Alphanumeric and hyphens only
- Cannot start or end with hyphen

### Environment Variable Keys
- Must follow POSIX naming conventions
- Maximum 256 characters
- Alphanumeric and underscores only
- Cannot start with digit
- Cannot override critical system variables (PATH, HOME, etc.)

### Secret Values
- Maximum 1MB size
- No null bytes or control characters
- Entropy checking for sensitive keys
- Pattern detection for dangerous content

### File Paths
- Must be absolute paths for secrets
- No path traversal sequences (..)
- Cannot access forbidden system directories
- Maximum 4KB length

## Performance Considerations

### Validation Efficiency
- Regex compilation cached in validators
- Early termination on validation failures
- Minimal memory allocation for checks
- Efficient string processing

### Resource Limits
- Size limits prevent excessive memory usage
- Timeout handling for command execution
- Bounded iteration over collections
- Safe integer conversions

## Integration with Other Packages

### Logging Integration
- All validation errors logged appropriately
- Sensitive data sanitized before logging
- Security events logged at appropriate levels
- Structured logging with context

### Configuration Integration
- Validates configuration file contents
- Checks environment variable definitions
- Validates repository URLs and paths
- Ensures secure default values

### Secret Management Integration
- Validates Docker Compose secrets
- Checks Podman secret configurations
- Validates secret target paths
- Ensures secure file permissions

## Mock Testing Support

### Command Runner Mocking
```go
// For testing system requirements without actual system calls
func TestSystemRequirements(t *testing.T) {
    mockRunner := &MockCommandRunner{
        responses: map[string][]byte{
            "systemctl --version": []byte("systemd 245"),
            "podman --version": []byte("podman version 3.4.0"),
        },
    }
    
    validate.SetCommandRunner(mockRunner)
    defer validate.ResetCommandRunner()
    
    err := validate.SystemRequirements()
    assert.NoError(t, err)
}
```

### Validator Testing
```go
// Test secret validation with various inputs
func TestSecretValidation(t *testing.T) {
    validator := validate.NewSecretValidator()
    
    // Test valid secret name
    err := validator.ValidateSecretName("my-secret")
    assert.NoError(t, err)
    
    // Test invalid secret name
    err = validator.ValidateSecretName("invalid_secret_name")
    assert.Error(t, err)
}
```
