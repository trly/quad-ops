//go:build darwin

package launchd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, DomainUser, opts.Domain)
	assert.Equal(t, "dev.trly.quad-ops", opts.LabelPrefix)
	assert.NotZero(t, opts.UID)
	assert.False(t, opts.UseSudo)

	homeDir, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(homeDir, "Library", "LaunchAgents"), opts.PlistDir)
	assert.Equal(t, filepath.Join(homeDir, "Library", "Logs", "quad-ops"), opts.LogsDir)
}

func TestOptionsFromSettings(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	t.Run("user mode with defaults", func(t *testing.T) {
		opts := OptionsFromSettings("", "", true)

		assert.Equal(t, DomainUser, opts.Domain)
		assert.Equal(t, "dev.trly.quad-ops", opts.LabelPrefix)
		assert.Equal(t, filepath.Join(homeDir, "Library", "LaunchAgents"), opts.PlistDir)
		assert.Equal(t, filepath.Join(homeDir, "Library", "Logs", "quad-ops"), opts.LogsDir)
		assert.False(t, opts.UseSudo)
	})

	t.Run("user mode with custom plist dir", func(t *testing.T) {
		customPlistDir := "/custom/plist/dir"
		opts := OptionsFromSettings("", customPlistDir, true)

		assert.Equal(t, DomainUser, opts.Domain)
		assert.Equal(t, customPlistDir, opts.PlistDir)
	})

	t.Run("system mode with defaults", func(t *testing.T) {
		opts := OptionsFromSettings("", "", false)

		assert.Equal(t, DomainSystem, opts.Domain)
		assert.Equal(t, "dev.trly.quad-ops", opts.LabelPrefix)
		assert.Equal(t, "/Library/LaunchDaemons", opts.PlistDir)
		assert.Equal(t, "/var/log/quad-ops", opts.LogsDir)
	})

	t.Run("system mode with custom plist dir", func(t *testing.T) {
		customPlistDir := "/custom/system/plist"
		opts := OptionsFromSettings("", customPlistDir, false)

		assert.Equal(t, DomainSystem, opts.Domain)
		assert.Equal(t, customPlistDir, opts.PlistDir)
		assert.Equal(t, "/var/log/quad-ops", opts.LogsDir)
	})
}

func TestOptions_Validate(t *testing.T) {
	t.Run("sets defaults for empty options", func(t *testing.T) {
		opts := Options{}
		err := opts.Validate()

		assert.Equal(t, DomainUser, opts.Domain)
		assert.Equal(t, "dev.trly.quad-ops", opts.LabelPrefix)

		if err == nil {
			t.Skip("podman is available, skipping error test")
		}
	})

	t.Run("validates invalid domain", func(t *testing.T) {
		opts := Options{
			Domain: "invalid",
		}
		err := opts.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain: invalid")
	})

	t.Run("sets user domain defaults", func(t *testing.T) {
		homeDir, _ := os.UserHomeDir()
		opts := Options{
			Domain:     DomainUser,
			PodmanPath: "/usr/bin/podman",
		}

		if err := opts.Validate(); err != nil {
			t.Skipf("skipping: podman not available: %v", err)
		}

		assert.Equal(t, filepath.Join(homeDir, "Library", "LaunchAgents"), opts.PlistDir)
		assert.Equal(t, filepath.Join(homeDir, "Library", "Logs", "quad-ops"), opts.LogsDir)
	})

	t.Run("system domain is not supported on macOS", func(t *testing.T) {
		opts := Options{
			Domain:     DomainSystem,
			PodmanPath: "/usr/bin/podman",
			PlistDir:   "/Library/LaunchDaemons",
			LogsDir:    "/var/log/quad-ops",
		}

		err := opts.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system domain launchd is not supported with rootless Podman on macOS")
	})

	t.Run("system domain is not supported", func(t *testing.T) {
		opts := Options{
			Domain:     DomainSystem,
			PodmanPath: "/usr/bin/podman",
		}

		err := opts.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system domain launchd is not supported")
	})

	t.Run("rejects invalid podman path", func(t *testing.T) {
		opts := Options{
			Domain:     DomainUser,
			PodmanPath: "/nonexistent/podman",
		}

		err := opts.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "podman binary not found")
	})
}

func TestResolvePodmanPath(t *testing.T) {
	path, err := resolvePodmanPath()

	if err != nil {
		t.Skipf("podman not found (this is expected if podman is not installed): %v", err)
	}

	assert.NotEmpty(t, path)
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "resolved path should exist")
}

func TestOptions_DomainID(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		expected string
	}{
		{
			name: "system domain",
			opts: Options{
				Domain: DomainSystem,
				UID:    501,
			},
			expected: "system",
		},
		{
			name: "user domain",
			opts: Options{
				Domain: DomainUser,
				UID:    501,
			},
			expected: "gui/501",
		},
		{
			name: "user domain with different UID",
			opts: Options{
				Domain: DomainUser,
				UID:    1000,
			},
			expected: "gui/1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.DomainID()
			assert.Equal(t, tt.expected, got)
		})
	}
}
