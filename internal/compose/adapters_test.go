package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
)

func TestFileSystemAdapterWithConfigProvider(t *testing.T) {
	// Create a test config provider
	testConfig := &config.Settings{
		QuadletDir: "/test/custom/quadlet/dir",
	}
	configProvider := config.NewDefaultConfigProvider()
	configProvider.SetConfig(testConfig)

	// Create filesystem adapter with config provider
	fsAdapter := NewFileSystemAdapterWithConfig(configProvider)

	// Test that the adapter uses the injected config
	unitPath := fsAdapter.GetUnitFilePath("test-service", "container")
	expected := "/test/custom/quadlet/dir/test-service.container"
	assert.Equal(t, expected, unitPath, "Adapter should use injected config for unit path")

	// Test GetContentHash functionality
	content := "test content"
	hash := fsAdapter.GetContentHash(content)
	assert.NotEmpty(t, hash, "Should return a non-empty hash")
}

func TestFileSystemAdapterBackwardCompatibility(t *testing.T) {
	// Create a default config provider for backward compatibility
	configProvider := config.NewDefaultConfigProvider()
	configProvider.InitConfig()

	// Create filesystem adapter with default config
	fsAdapter := NewFileSystemAdapterWithConfig(configProvider)

	// Should still work using default config
	unitPath := fsAdapter.GetUnitFilePath("test-service", "container")
	assert.Contains(t, unitPath, "test-service.container", "Should generate unit path using default config")

	// Test GetContentHash functionality
	content := "test content"
	hash := fsAdapter.GetContentHash(content)
	assert.NotEmpty(t, hash, "Should return a non-empty hash")
}
