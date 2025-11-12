// Package validate provides security validation functions for sensitive data
package validate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/trly/quad-ops/internal/log"
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
	logger       log.Logger
}

// NewSecretValidator creates a new SecretValidator instance.
func NewSecretValidator(logger log.Logger) *SecretValidator {
	// Allow printable ASCII characters, excluding control characters
	allowedChars := regexp.MustCompile(`^[\x20-\x7E\s]*$`)
	return &SecretValidator{
		allowedChars: allowedChars,
		logger:       logger,
	}
}

// ValidateSecretName validates that a secret name is safe and follows conventions.
func (sv *SecretValidator) ValidateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	if len(name) > MaxSecretNameLen {
		sv.logger.Warn("Secret name is very long, consider shortening", "name", name, "length", len(name), "max_recommended", MaxSecretNameLen)
	}

	// Docker Compose allows flexible secret names, only warn about non-DNS conventions
	dnsPattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?)*$`)
	if !dnsPattern.MatchString(name) {
		sv.logger.Warn("Secret name doesn't follow DNS naming conventions", "name", name)
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

	// Warn about potentially dangerous patterns (compose spec allows any content)
	lowerValue := strings.ToLower(value)
	for _, pattern := range securityPatterns {
		if strings.Contains(lowerValue, strings.ToLower(pattern)) {
			sv.logger.Warn("Secret value contains potentially sensitive pattern", "pattern", pattern)
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
		sv.logger.Warn("Secret target path is very long", "target", target, "length", len(target), "max_recommended", MaxSecretTargetLen)
	}

	// Ensure target is an absolute path for security
	if !strings.HasPrefix(target, "/") {
		return fmt.Errorf("secret target must be an absolute path: %s", target)
	}

	// Prevent path traversal attacks
	if strings.Contains(target, "..") {
		return fmt.Errorf("secret target contains path traversal sequence: %s", target)
	}

	// Warn about potentially risky system directories (compose spec allows any path)
	sensitivePaths := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts", "/proc", "/sys"}
	for _, sensitive := range sensitivePaths {
		if strings.HasPrefix(target, sensitive) {
			sv.logger.Warn("Secret target path may access sensitive system directory", "target", target, "sensitive_path", sensitive)
		}
	}

	return nil
}

// ValidateEnvValue validates environment variable values for size and content.
func (sv *SecretValidator) ValidateEnvValue(key, value string) error {
	if len(value) > MaxEnvValueSize {
		sv.logger.Warn("Environment variable value is very large", "key", key, "size", len(value), "max_recommended", MaxEnvValueSize)
	}

	// Check for null bytes - this is a real compose limitation
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("environment variable value contains null byte")
	}

	// If this appears to be a sensitive variable, apply stricter validation as warnings
	if isSensitiveKey(key) {
		sv.warnSensitiveEnvValue(key, value)
	}

	return nil
}

// warnSensitiveEnvValue warns about potentially weak sensitive environment variables.
func (sv *SecretValidator) warnSensitiveEnvValue(key, value string) {
	// Warn about obvious test/default values
	testValues := []string{"password", "secret", "123456", "admin", "test", "default"}
	lowerValue := strings.ToLower(strings.TrimSpace(value))

	for _, testValue := range testValues {
		if lowerValue == testValue {
			sv.logger.Warn("Sensitive environment variable appears to contain a test/default value", "key", key)
			return
		}
	}

	// Warn about short sensitive values
	if len(value) < 8 {
		sv.logger.Warn("Sensitive environment variable value is short, consider using stronger value", "key", key, "length", len(value))
	}

	// Warn about repeating patterns
	if isRepeatingPattern(value) {
		sv.logger.Warn("Sensitive environment variable value has low entropy", "key", key)
	}
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

// ValidateEnvKey provides extended validation for environment variable keys.
func (sv *SecretValidator) ValidateEnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	if len(key) > MaxEnvKeyLen {
		sv.logger.Warn("Environment variable key is very long", "key", key, "length", len(key), "max_recommended", MaxEnvKeyLen)
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

// EnvKey provides extended validation for environment variable keys.
// Deprecated: Use SecretValidator.ValidateEnvKey instead.
func EnvKey(key string) error {
	logger := log.NewLogger(false)
	validator := NewSecretValidator(logger)
	return validator.ValidateEnvKey(key)
}

// UnitName validates that a unit name is safe for use in shell commands.
// Unit names must follow systemd naming conventions to prevent command injection.
func UnitName(unitName string) error {
	if unitName == "" {
		return fmt.Errorf("unit name cannot be empty")
	}

	// Systemd unit names must match this pattern: alphanumeric, dots, dashes, underscores, @, and colons
	// This prevents injection of shell metacharacters like ;, |, &, $, etc.
	validUnitName := regexp.MustCompile(`^[a-zA-Z0-9._@:-]+$`)
	if !validUnitName.MatchString(unitName) {
		return fmt.Errorf("invalid unit name: contains unsafe characters")
	}

	// Additional length check to prevent extremely long names
	if len(unitName) > 256 {
		return fmt.Errorf("unit name too long")
	}

	return nil
}

// Path validates that a path doesn't contain path traversal sequences.
// It uses filepath.Clean to normalize the path and checks for traversal attempts.
func Path(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to normalize it and resolve any traversal sequences
	cleanPath := filepath.Clean(path)

	// If the cleaned path is different and contains traversal, it's suspicious
	if cleanPath != path && strings.Contains(path, "..") {
		return fmt.Errorf("path contains path traversal sequence")
	}

	// Check if the cleaned path tries to go above the current directory for relative paths
	if !filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path attempts to traverse above working directory")
	}

	return nil
}

// PathWithinBase ensures a path stays within a base directory after cleaning.
// This is more secure than Path alone for critical file operations.
func PathWithinBase(path, basePath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	// Clean both paths to normalize them
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(basePath)

	// Make paths absolute for proper comparison
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(absBase, cleanPath)
	}

	// Clean the final path
	absPath = filepath.Clean(absPath)

	// Ensure the final path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path escapes base directory")
	}

	return absPath, nil
}
