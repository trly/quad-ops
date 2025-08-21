package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/log"
)

func TestSecretValidator_ValidateSecretName(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(false)
	validator := NewSecretValidator(logger)

	tests := []struct {
		name        string
		secretName  string
		expectError bool
	}{
		{"valid simple name", "my-secret", false},
		{"valid with subdomain", "app.my-secret", false},
		{"empty name", "", true},
		{"too long name", strings.Repeat("a", 254), false}, // Now just warns, doesn't error
		{"invalid chars", "my_secret!", false},             // Now just warns, doesn't error
		{"starts with number", "1secret", false},           // DNS allows starting with number
		{"uppercase chars", "MySecret", false},             // Now just warns, doesn't error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSecretName(tt.secretName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretValidator_ValidateSecretValue(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(false)
	validator := NewSecretValidator(logger)

	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"valid simple value", "mysecretvalue123", false},
		{"valid with special chars", "my!@#$%^secret", false},
		{"empty value", "", true},
		{"too large value", strings.Repeat("a", MaxSecretFileSize+1), true},
		{"null byte", "secret\x00value", true},
		{"control character", "secret\x01value", true},
		{"dangerous pattern", "BEGIN PRIVATE KEY", false}, // Now just warns, doesn't error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSecretValue(tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretValidator_ValidateSecretTarget(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(false)
	validator := NewSecretValidator(logger)

	tests := []struct {
		name        string
		target      string
		expectError bool
	}{
		{"valid absolute path", "/run/secrets/mysecret", false},
		{"valid nested path", "/app/secrets/config.json", false},
		{"empty target", "", true},
		{"relative path", "secrets/mysecret", true},
		{"path traversal", "/run/secrets/../../../etc/passwd", true},
		{"system path", "/etc/passwd", false},                                               // Now just warns, doesn't error
		{"proc path", "/proc/version", false},                                               // Now just warns, doesn't error
		{"too long path", "/run/secrets/" + strings.Repeat("a", MaxSecretTargetLen), false}, // Now just warns, doesn't error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSecretTarget(tt.target)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretValidator_ValidateEnvValue(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(false)
	validator := NewSecretValidator(logger)

	tests := []struct {
		name        string
		key         string
		value       string
		expectError bool
	}{
		{"regular env var", "DEBUG", "true", false},
		{"large non-sensitive value", "LARGE_CONFIG", strings.Repeat("a", 1000), false},
		{"sensitive short value", "PASSWORD", "123", false},   // Now just warns, doesn't error
		{"sensitive test value", "SECRET", "password", false}, // Now just warns, doesn't error
		{"sensitive valid value", "API_KEY", "sk_live_abcdef123456789", false},
		{"too large value", "CONFIG", strings.Repeat("a", MaxEnvValueSize+1), false}, // Now just warns, doesn't error
		{"null byte", "VALUE", "test\x00value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEnvValue(tt.key, tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEnvKey(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(false)
	validator := NewSecretValidator(logger)

	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{"valid uppercase", "MY_VAR", false},
		{"valid mixed case", "MyApp_Config", false},
		{"valid with numbers", "VAR123", false},
		{"empty key", "", true},
		{"starts with digit", "1VAR", true},
		{"invalid character", "MY-VAR", true},
		{"too long key", strings.Repeat("A", MaxEnvKeyLen+1), false}, // Now just warns, doesn't error
		{"starts with underscore", "_PRIVATE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEnvKey(tt.key)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeForLogging(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{"non-sensitive key", "DEBUG", "true", "true"},
		{"sensitive password", "PASSWORD", "secret123", "se*****23"},
		{"sensitive api key", "API_KEY", "abcdef", "ab**ef"},
		{"short sensitive", "SECRET", "abc", "[REDACTED]"},
		{"long password", "USER_PASSWORD", "verylongsecretpassword", "ve******************rd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLogging(tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		sensitive bool
	}{
		{"password key", "USER_PASSWORD", true},
		{"secret key", "API_SECRET", true},
		{"token key", "ACCESS_TOKEN", true},
		{"normal key", "DEBUG_MODE", false},
		{"config key", "APP_CONFIG", false},
		{"key with key", "ENCRYPTION_KEY", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveKey(tt.key)
			assert.Equal(t, tt.sensitive, result)
		})
	}
}

func TestIsRepeatingPattern(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		repeating bool
	}{
		{"normal string", "abcdef123", false},
		{"repeating chars", "aaaaaaaa", true},
		{"mixed repeating", "abababab", false}, // This test might need adjustment based on implementation
		{"short string", "abc", false},
		{"mostly same char", "aaabaaaa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRepeatingPattern(tt.value)
			assert.Equal(t, tt.repeating, result)
		})
	}
}
