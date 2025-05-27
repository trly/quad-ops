// Package validate provides security validation functions for sensitive data
package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Security constants for size limits and validation.
const (
	MaxSecretFileSize  = 1024 * 1024 // 1MB - Maximum size for secret files
	MaxEnvValueSize    = 32768       // 32KB - Maximum size for environment variable values
	MaxSecretNameLen   = 253         // DNS name limit - Maximum length for secret names
	MaxEnvKeyLen       = 256         // Maximum length for environment variable keys
	MaxSecretTargetLen = 4096        // Maximum length for secret target paths
)

var (
	// securityPatterns contains patterns that should not appear in secret values.
	securityPatterns = []string{
		"BEGIN PRIVATE KEY",
		"BEGIN RSA PRIVATE KEY",
		"BEGIN CERTIFICATE",
		"sql",
		"exec",
		"system",
	}

	// sensitiveKeywords identifies potentially sensitive environment variable names.
	sensitiveKeywords = []string{
		"password", "secret", "key", "token", "auth", "credential",
		"private", "cert", "ssl", "tls", "api_key", "access_key",
	}
)

// SecretValidator provides validation for secrets and sensitive data.
type SecretValidator struct {
	allowedChars *regexp.Regexp
}

// NewSecretValidator creates a new SecretValidator instance.
func NewSecretValidator() *SecretValidator {
	// Allow printable ASCII characters, excluding control characters
	allowedChars := regexp.MustCompile(`^[\x20-\x7E\s]*$`)
	return &SecretValidator{
		allowedChars: allowedChars,
	}
}

// ValidateSecretName validates that a secret name is safe and follows conventions.
func (sv *SecretValidator) ValidateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	if len(name) > MaxSecretNameLen {
		return fmt.Errorf("secret name too long: %d characters (max %d)", len(name), MaxSecretNameLen)
	}

	// Secret names should follow DNS naming conventions
	dnsPattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?)*$`)
	if !dnsPattern.MatchString(name) {
		return fmt.Errorf("secret name must follow DNS naming conventions: %s", name)
	}

	return nil
}

// ValidateSecretValue validates that a secret value is safe and within size limits.
func (sv *SecretValidator) ValidateSecretValue(value string) error {
	if len(value) == 0 {
		return fmt.Errorf("secret value cannot be empty")
	}

	if len(value) > MaxSecretFileSize {
		return fmt.Errorf("secret value too large: %d bytes (max %d)", len(value), MaxSecretFileSize)
	}

	// Check for null bytes and other control characters that could cause issues
	for i, r := range value {
		if r == 0 {
			return fmt.Errorf("secret value contains null byte at position %d", i)
		}
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("secret value contains control character at position %d", i)
		}
	}

	// Check for potentially dangerous patterns
	lowerValue := strings.ToLower(value)
	for _, pattern := range securityPatterns {
		if strings.Contains(lowerValue, strings.ToLower(pattern)) {
			return fmt.Errorf("secret value contains potentially dangerous pattern: %s", pattern)
		}
	}

	return nil
}

// ValidateSecretTarget validates that a secret target path is safe.
func (sv *SecretValidator) ValidateSecretTarget(target string) error {
	if target == "" {
		return fmt.Errorf("secret target cannot be empty")
	}

	if len(target) > MaxSecretTargetLen {
		return fmt.Errorf("secret target path too long: %d characters (max %d)", len(target), MaxSecretTargetLen)
	}

	// Ensure target is an absolute path for security
	if !strings.HasPrefix(target, "/") {
		return fmt.Errorf("secret target must be an absolute path: %s", target)
	}

	// Prevent path traversal attacks
	if strings.Contains(target, "..") {
		return fmt.Errorf("secret target contains path traversal sequence: %s", target)
	}

	// Prevent access to sensitive system directories
	forbiddenPaths := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts", "/proc", "/sys"}
	for _, forbidden := range forbiddenPaths {
		if strings.HasPrefix(target, forbidden) {
			return fmt.Errorf("secret target accesses forbidden system path: %s", target)
		}
	}

	return nil
}

// ValidateEnvValue validates environment variable values for size and content.
func (sv *SecretValidator) ValidateEnvValue(key, value string) error {
	if len(value) > MaxEnvValueSize {
		return fmt.Errorf("environment variable value too large: %d bytes (max %d)", len(value), MaxEnvValueSize)
	}

	// Check for null bytes
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("environment variable value contains null byte")
	}

	// If this appears to be a sensitive variable, apply stricter validation
	if isSensitiveKey(key) {
		return sv.validateSensitiveEnvValue(value)
	}

	return nil
}

// validateSensitiveEnvValue applies stricter validation for sensitive environment variables.
func (sv *SecretValidator) validateSensitiveEnvValue(value string) error {
	// Sensitive values should not contain obvious test/default values
	testValues := []string{"password", "secret", "123456", "admin", "test", "default"}
	lowerValue := strings.ToLower(strings.TrimSpace(value))

	for _, testValue := range testValues {
		if lowerValue == testValue {
			return fmt.Errorf("sensitive environment variable appears to contain a test/default value")
		}
	}

	// Sensitive values should have minimum entropy (basic check)
	if len(value) < 8 {
		return fmt.Errorf("sensitive environment variable value is too short (minimum 8 characters)")
	}

	// Check for repeating characters (e.g., "aaaaaaaa")
	if isRepeatingPattern(value) {
		return fmt.Errorf("sensitive environment variable value has low entropy")
	}

	return nil
}

// isSensitiveKey checks if an environment variable key indicates sensitive data.
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerKey, keyword) {
			return true
		}
	}
	return false
}

// isRepeatingPattern checks if a string consists mostly of repeating characters.
func isRepeatingPattern(s string) bool {
	if len(s) < 4 {
		return false
	}

	// Count character frequencies
	charCount := make(map[rune]int)
	for _, r := range s {
		charCount[r]++
	}

	// If any character appears more than 50% of the time, consider it repeating.
	threshold := len(s) / 2
	for _, count := range charCount {
		if count > threshold {
			return true
		}
	}

	return false
}

// SanitizeForLogging redacts sensitive information from strings for safe logging.
func SanitizeForLogging(key, value string) string {
	if isSensitiveKey(key) {
		if len(value) <= 4 {
			return "[REDACTED]"
		}
		// Show first 2 and last 2 characters with asterisks in between
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	}
	return value
}

// EnvKey provides extended validation for environment variable keys.
func EnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	if len(key) > MaxEnvKeyLen {
		return fmt.Errorf("environment variable key too long: %d characters (max %d)", len(key), MaxEnvKeyLen)
	}

	// Environment variable names should follow POSIX conventions
	// Allow alphanumeric characters and underscores, but not start with digits
	for i, r := range key {
		if i == 0 {
			if unicode.IsDigit(r) {
				return fmt.Errorf("environment variable key cannot start with digit: %s", key)
			}
			if !unicode.IsLetter(r) && r != '_' {
				return fmt.Errorf("environment variable key must start with letter or underscore: %s", key)
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return fmt.Errorf("environment variable key contains invalid character '%c': %s", r, key)
			}
		}
	}

	return nil
}
