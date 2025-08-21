package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

func TestGetUnitFilePath(t *testing.T) {
	// Set up config for testing
	cfg := &config.Settings{
		QuadletDir: "/test/quadlet",
	}
	configProvider := config.NewDefaultConfigProvider()
	configProvider.SetConfig(cfg)

	tests := []struct {
		name     string
		unitName string
		unitType string
		expected string
	}{
		{
			name:     "container unit",
			unitName: "test-service",
			unitType: "container",
			expected: "/test/quadlet/test-service.container",
		},
		{
			name:     "volume unit",
			unitName: "test-volume",
			unitType: "volume",
			expected: "/test/quadlet/test-volume.volume",
		},
		{
			name:     "network unit",
			unitName: "test-network",
			unitType: "network",
			expected: "/test/quadlet/test-network.network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service with mock config
			cfg := &config.Settings{QuadletDir: "/test/quadlet"}
			provider := &config.MockProvider{Config: cfg}
			service := NewServiceWithLogger(provider, log.NewLogger(false))
			result := service.GetUnitFilePath(tt.unitName, tt.unitType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasUnitChanged(t *testing.T) {
	// Use no-op logger for testing
	logger := log.Nop()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck // Test cleanup

	tests := []struct {
		name            string
		existingContent string
		newContent      string
		fileExists      bool
		expected        bool
	}{
		{
			name:            "file doesn't exist",
			existingContent: "",
			newContent:      "new content",
			fileExists:      false,
			expected:        true,
		},
		{
			name:            "content unchanged",
			existingContent: "same content",
			newContent:      "same content",
			fileExists:      true,
			expected:        false,
		},
		{
			name:            "content changed",
			existingContent: "old content",
			newContent:      "new content",
			fileExists:      true,
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unitPath := filepath.Join(tempDir, "test.container")

			if tt.fileExists {
				err := os.WriteFile(unitPath, []byte(tt.existingContent), 0600)
				require.NoError(t, err)
			}

			// Create service with temp directory config
			cfg := &config.Settings{QuadletDir: tempDir}
			provider := &config.MockProvider{Config: cfg}
			service := NewServiceWithLogger(provider, logger)
			result := service.HasUnitChanged(unitPath, tt.newContent)
			assert.Equal(t, tt.expected, result)

			// Clean up for next test
			if tt.fileExists {
				os.Remove(unitPath) //nolint:errcheck,gosec // Test cleanup
			}
		})
	}
}

func TestWriteUnitFile(t *testing.T) {
	// Use no-op logger for testing
	logger := log.Nop()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck // Test cleanup

	tests := []struct {
		name        string
		unitPath    string
		content     string
		expectError bool
	}{
		{
			name:        "successful write",
			unitPath:    filepath.Join(tempDir, "test.container"),
			content:     "test content",
			expectError: false,
		},
		{
			name:        "write with subdirectory creation",
			unitPath:    filepath.Join(tempDir, "subdir", "test.container"),
			content:     "test content",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service with temp directory config
			cfg := &config.Settings{QuadletDir: tempDir}
			provider := &config.MockProvider{Config: cfg}
			service := NewServiceWithLogger(provider, logger)
			err := service.WriteUnitFile(tt.unitPath, tt.content)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify file was written correctly
				writtenContent, err := os.ReadFile(tt.unitPath)
				require.NoError(t, err)
				assert.Equal(t, tt.content, string(writtenContent))

				// Verify file permissions
				info, err := os.Stat(tt.unitPath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
			}
		})
	}
}

func TestGetContentHash(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:     "simple content",
			content:  "hello world",
			expected: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetContentHash(tt.content)
			assert.Equal(t, tt.expected, fmt.Sprintf("%x", result))
		})
	}
}

func TestServiceWithConfigProvider(t *testing.T) {
	// Create a test config provider
	testConfig := &config.Settings{
		QuadletDir: "/test/custom/quadlet/dir",
	}
	configProvider := config.NewDefaultConfigProvider()
	configProvider.SetConfig(testConfig)

	// Create filesystem service with config provider
	fsService := NewService(configProvider)

	// Test GetUnitFilePath uses the injected config
	unitPath := fsService.GetUnitFilePath("test-service", "container")
	expected := "/test/custom/quadlet/dir/test-service.container"
	assert.Equal(t, expected, unitPath, "Service should use injected config for unit path")

	// Test GetUnitFilesDirectory uses the injected config
	dir := fsService.GetUnitFilesDirectory()
	assert.Equal(t, "/test/custom/quadlet/dir", dir, "Service should use injected config for quadlet directory")

	// Test GetContentHash works the same way
	content := "test content"
	hash1 := fsService.GetContentHash(content)
	hash2 := string(GetContentHash(content)) // Compare with legacy function
	assert.Equal(t, hash1, hash2, "Service hash should match legacy hash function")
}
