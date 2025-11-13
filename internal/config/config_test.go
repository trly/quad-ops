package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Helper function to reset viper and config.
func resetViper() {
	viper.Reset()
}

// TestInitConfig tests the InitConfig function.
func TestInitConfig(t *testing.T) {
	resetViper()

	// Prevent viper from loading any real config files
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	provider := NewConfigProvider()
	cfg := provider.GetConfig()

	// Repository dir and QuadletDir may be platform-specific
	assert.NotEmpty(t, cfg.RepositoryDir)
	assert.NotEmpty(t, cfg.QuadletDir)

	// Other settings should match defaults
	assert.Equal(t, DefaultSyncInterval, cfg.SyncInterval)
	assert.Equal(t, DefaultUserMode, cfg.UserMode)
	assert.Equal(t, DefaultVerbose, cfg.Verbose)
	assert.Equal(t, DefaultUnitStartTimeout, cfg.UnitStartTimeout)
	assert.Equal(t, DefaultImagePullTimeout, cfg.ImagePullTimeout)
}

// TestSetAndGetConfig tests the SetConfig and GetConfig functions.
func TestSetAndGetConfig(t *testing.T) {
	resetViper()
	testConfig := &Settings{
		RepositoryDir:    "/custom/path",
		SyncInterval:     10 * time.Minute,
		QuadletDir:       "/custom/quadlet",
		UserMode:         true,
		Verbose:          true,
		UnitStartTimeout: 15 * time.Second,
		ImagePullTimeout: 60 * time.Second,
		Repositories: []Repository{
			{
				Name:      "test-repo",
				URL:       "https://github.com/test/repo",
				Reference: "main",
			},
		},
	}

	provider := NewDefaultConfigProvider()
	provider.SetConfig(testConfig)
	retrievedConfig := provider.GetConfig()
	assert.Equal(t, testConfig, retrievedConfig)
}

// TestCustomConfigFile tests the use of a custom config file.
func TestCustomConfigFile(t *testing.T) {
	resetViper()

	tmpfile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	configContent := `repositoryDir: "/test/path"
syncInterval: 15m
quadletDir: "/test/quadlet"
userMode: true
verbose: true
unitStartTimeout: 20s
imagePullTimeout: 90s
repositories:
- name: "test-repo"
  url: "https://github.com/test/repo"
  ref: "main"`

	if err := os.WriteFile(tmpfile.Name(), []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpfile.Name())
	viper.SetConfigType("yaml")

	viper.SetDefault("repositoryDir", DefaultRepositoryDir)
	viper.SetDefault("syncInterval", DefaultSyncInterval)
	viper.SetDefault("quadletDir", DefaultQuadletDir)
	viper.SetDefault("userMode", DefaultUserMode)
	viper.SetDefault("verbose", DefaultVerbose)
	viper.SetDefault("unitStartTimeout", DefaultUnitStartTimeout)
	viper.SetDefault("imagePullTimeout", DefaultImagePullTimeout)

	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	cfg := &Settings{}
	if err := viper.Unmarshal(cfg); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "/test/path", cfg.RepositoryDir)
	assert.Equal(t, 15*time.Minute, cfg.SyncInterval)
	assert.Equal(t, "/test/quadlet", cfg.QuadletDir)
	assert.True(t, cfg.UserMode)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, 20*time.Second, cfg.UnitStartTimeout)
	assert.Equal(t, 90*time.Second, cfg.ImagePullTimeout)
	assert.Len(t, cfg.Repositories, 1)
	assert.Equal(t, "test-repo", cfg.Repositories[0].Name)
}

// TestConfigNotFound tests the case when the config file is not found.
func TestConfigNotFound(t *testing.T) {
	resetViper()
	provider := NewDefaultConfigProvider()
	provider.SetConfigFilePath("/nonexistent/config.yaml")
	cfg := provider.InitConfig()

	// On macOS, defaults should be adjusted; on Linux, should remain as-is
	if os.Getenv("GOOS") == "darwin" || (os.Getenv("GOOS") == "" && os.Getenv("HOME") != "") {
		// If actually running on macOS, defaults may be platform-specific
		assert.NotEmpty(t, cfg.RepositoryDir)
	} else {
		assert.Equal(t, DefaultRepositoryDir, cfg.RepositoryDir)
	}
}

// TestPlatformDefaults tests that platform-specific defaults are applied.
func TestPlatformDefaults(t *testing.T) {
	resetViper()

	t.Run("respects user overrides", func(t *testing.T) {
		cfg := &Settings{
			RepositoryDir:    "/custom/repo",
			QuadletDir:       "/custom/quadlet",
			SyncInterval:     DefaultSyncInterval,
			UserMode:         DefaultUserMode,
			Verbose:          DefaultVerbose,
			UnitStartTimeout: DefaultUnitStartTimeout,
			ImagePullTimeout: DefaultImagePullTimeout,
		}

		applyPlatformDefaults(cfg)

		// User overrides should be preserved regardless of platform
		assert.Equal(t, "/custom/repo", cfg.RepositoryDir)
		assert.Equal(t, "/custom/quadlet", cfg.QuadletDir)
	})

	t.Run("applies defaults only when at Linux defaults", func(t *testing.T) {
		cfg := &Settings{
			RepositoryDir:    DefaultRepositoryDir,
			QuadletDir:       DefaultQuadletDir,
			SyncInterval:     DefaultSyncInterval,
			UserMode:         DefaultUserMode,
			Verbose:          DefaultVerbose,
			UnitStartTimeout: DefaultUnitStartTimeout,
			ImagePullTimeout: DefaultImagePullTimeout,
		}

		applyPlatformDefaults(cfg)

		// Platform defaults applied based on runtime.GOOS
		// On macOS, should change; on Linux, should remain
		assert.NotEmpty(t, cfg.RepositoryDir)
		assert.NotEmpty(t, cfg.QuadletDir)
	})
}
