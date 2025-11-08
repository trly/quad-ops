//go:build darwin

package launchd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodePlist(t *testing.T) {
	t.Run("minimal plist", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			ProgramArguments: []string{"/usr/bin/example", "--arg"},
			RunAtLoad:        true,
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<?xml version=\"1.0\"")
		assert.Contains(t, result, "<key>Label</key>")
		assert.Contains(t, result, "<string>com.example.test</string>")
		assert.Contains(t, result, "<key>ProgramArguments</key>")
		assert.Contains(t, result, "<key>RunAtLoad</key>")
	})

	t.Run("plist with environment variables", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			ProgramArguments: []string{"/usr/bin/test"},
			EnvironmentVariables: map[string]string{
				"PATH": "/usr/local/bin:/usr/bin",
				"HOME": "/Users/test",
			},
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<key>EnvironmentVariables</key>")
		assert.Contains(t, result, "<key>PATH</key>")
	})

	t.Run("plist with KeepAlive bool", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			ProgramArguments: []string{"/usr/bin/test"},
			KeepAlive:        true,
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<key>KeepAlive</key>")
		assert.Contains(t, result, "<true/>")
	})

	t.Run("plist with KeepAlive dict", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			ProgramArguments: []string{"/usr/bin/test"},
			KeepAlive: map[string]bool{
				"SuccessfulExit": false,
				"Crashed":        true,
			},
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<key>KeepAlive</key>")
		assert.Contains(t, result, "<key>SuccessfulExit</key>")
		assert.Contains(t, result, "<key>Crashed</key>")
	})

	t.Run("plist with Program field only", func(t *testing.T) {
		p := &Plist{
			Label:   "com.example.test",
			Program: "/usr/bin/test",
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<key>Program</key>", "missing Program key")
		assert.Contains(t, result, "<string>/usr/bin/test</string>", "missing Program value")
		assert.NotContains(t, result, "<key>ProgramArguments</key>", "ProgramArguments should not be present")
	})

	t.Run("plist with ProgramArguments excludes Program", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			Program:          "/usr/bin/test", // Should be ignored
			ProgramArguments: []string{"/usr/bin/test", "--arg"},
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<key>ProgramArguments</key>", "missing ProgramArguments key")
		assert.NotContains(t, result, "<key>Program</key>", "Program key should not be present when ProgramArguments is used")
	})

	t.Run("plist with optional fields", func(t *testing.T) {
		p := &Plist{
			Label:             "com.example.test",
			ProgramArguments:  []string{"/usr/bin/test"},
			WorkingDirectory:  "/var/tmp",
			UserName:          "testuser",
			GroupName:         "testgroup",
			StandardOutPath:   "/var/log/test.log",
			StandardErrorPath: "/var/log/test.err",
			ThrottleInterval:  10,
			ProcessType:       "Interactive",
			SessionCreate:     true,
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		expectedKeys := []string{
			"WorkingDirectory",
			"UserName",
			"GroupName",
			"StandardOutPath",
			"StandardErrorPath",
			"ThrottleInterval",
			"ProcessType",
			"SessionCreate",
		}

		for _, key := range expectedKeys {
			assert.Contains(t, result, "<key>"+key+"</key>", "missing %s key", key)
		}
	})

	t.Run("plist with XML escaping", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			ProgramArguments: []string{"/usr/bin/test", "--arg=<value>", "foo&bar"},
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		if strings.Contains(result, "<value>") {
			assert.Contains(t, result, "&lt;value&gt;", "XML special characters not escaped properly")
		}
		if strings.Contains(result, "foo&bar") {
			assert.Contains(t, result, "foo&amp;bar", "ampersand not escaped")
		}
	})

	t.Run("plist with service dependencies", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.service",
			ProgramArguments: []string{"/usr/bin/service"},
			DependsOn: []string{
				"com.example.database",
				"com.example.cache",
			},
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.Contains(t, result, "<key>DependsOn</key>", "missing DependsOn key")
		assert.Contains(t, result, "<string>com.example.database</string>", "missing database dependency")
		assert.Contains(t, result, "<string>com.example.cache</string>", "missing cache dependency")
	})

	t.Run("plist with empty dependencies", func(t *testing.T) {
		p := &Plist{
			Label:            "com.example.test",
			ProgramArguments: []string{"/usr/bin/test"},
			DependsOn:        []string{},
		}

		data, err := EncodePlist(p)
		require.NoError(t, err)

		result := string(data)
		assert.NotContains(t, result, "<key>DependsOn</key>", "DependsOn should not be present for empty list")
	})
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "alphanumeric",
			input:    "com.example.MyApp123",
			expected: "com.example.MyApp123",
		},
		{
			name:     "with dashes and underscores",
			input:    "com.github.my-app_test",
			expected: "com.github.my-app_test",
		},
		{
			name:     "with spaces",
			input:    "my app service",
			expected: "my-app-service",
		},
		{
			name:     "with special characters",
			input:    "app@service!test",
			expected: "app-service-test",
		},
		{
			name:     "unicode characters",
			input:    "service™®",
			expected: "service--",
		},
		{
			name:     "mixed special chars",
			input:    "com.example/my_app:v1",
			expected: "com.example-my_app-v1",
		},
		{
			name:     "already clean",
			input:    "dev.trly.quad-ops.web-service",
			expected: "dev.trly.quad-ops.web-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLabel(tt.input)
			assert.Equal(t, tt.expected, got, "SanitizeLabel(%q)", tt.input)
		})
	}
}

func TestWriteHelpers(t *testing.T) {
	t.Run("writeDictEntry skips empty values", func(t *testing.T) {
		var buf bytes.Buffer
		writeDictEntry(&buf, "TestKey", "")
		assert.Equal(t, 0, buf.Len(), "writeDictEntry should skip empty values")
	})

	t.Run("writeDictIntEntry skips zero values", func(t *testing.T) {
		var buf bytes.Buffer
		writeDictIntEntry(&buf, "TestKey", 0)
		assert.Equal(t, 0, buf.Len(), "writeDictIntEntry should skip zero values")
	})

	t.Run("writeDictArrayEntry skips empty arrays", func(t *testing.T) {
		var buf bytes.Buffer
		writeDictArrayEntry(&buf, "TestKey", []string{})
		assert.Equal(t, 0, buf.Len(), "writeDictArrayEntry should skip empty arrays")
	})

	t.Run("writeDictDictEntry skips empty maps", func(t *testing.T) {
		var buf bytes.Buffer
		writeDictDictEntry(&buf, "TestKey", map[string]string{})
		assert.Equal(t, 0, buf.Len(), "writeDictDictEntry should skip empty maps")
	})
}
