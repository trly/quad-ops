package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Helper function to reset viper and config
func resetViper() {
	viper.Reset()
	cfg = nil
}

// TestInitConfig tests the InitConfig function
func TestInitConfig(t *testing.T) {
	resetViper()
	cfg := InitConfig()
	assert.Equal(t, DefaultRepositoryDir, cfg.RepositoryDir)
	assert.Equal(t, DefaultSyncInterval, cfg.SyncInterval)
	assert.Equal(t, DefaultQuadletDir, cfg.QuadletDir)
	assert.Equal(t, DefaultDBPath, cfg.DBPath)
	assert.Equal(t, DefaultUserMode, cfg.UserMode)
	assert.Equal(t, DefaultVerbose, cfg.Verbose)
}

// TestSetAndGetConfig tests the SetConfig and GetConfig functions
func TestSetAndGetConfig(t *testing.T) {
	resetViper()
	testConfig := &Config{
		RepositoryDir: "/custom/path",
		SyncInterval:  10 * time.Minute,
		QuadletDir:    "/custom/quadlet",
		DBPath:        "/custom/db.sqlite",
		UserMode:      true,
		Verbose:       true,
		Repositories: []Repository{
			{
				Name:   "test-repo",
				URL:    "https://github.com/test/repo",
				Target: "main",
				Cleanup: CleanupPolicy{
					Action: "delete",
				},
			},
		},
	}

	SetConfig(testConfig)
	retrievedConfig := GetConfig()
	assert.Equal(t, testConfig, retrievedConfig)
}

// TestCustomConfigFile tests the use of a custom config file
func TestCustomConfigFile(t *testing.T) {
	resetViper()

	tmpfile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `repositoryDir: "/test/path"
syncInterval: 15m
quadletDir: "/test/quadlet"
dbPath: "/test/db.sqlite"
userMode: true
verbose: true
repositories:
- name: "test-repo"
  url: "https://github.com/test/repo"
  target: "main"
  cleanup:
    action: "delete"`

	if err := os.WriteFile(tmpfile.Name(), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpfile.Name())
	viper.SetConfigType("yaml")

	viper.SetDefault("repositoryDir", DefaultRepositoryDir)
	viper.SetDefault("syncInterval", DefaultSyncInterval)
	viper.SetDefault("quadletDir", DefaultQuadletDir)
	viper.SetDefault("dbPath", DefaultDBPath)
	viper.SetDefault("userMode", DefaultUserMode)
	viper.SetDefault("verbose", DefaultVerbose)

	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "/test/path", cfg.RepositoryDir)
	assert.Equal(t, 15*time.Minute, cfg.SyncInterval)
	assert.Equal(t, "/test/quadlet", cfg.QuadletDir)
	assert.Equal(t, "/test/db.sqlite", cfg.DBPath)
	assert.True(t, cfg.UserMode)
	assert.True(t, cfg.Verbose)
	assert.Len(t, cfg.Repositories, 1)
	assert.Equal(t, "test-repo", cfg.Repositories[0].Name)
}

// TestConfigNotFound tests the case when the config file is not found
func TestConfigNotFound(t *testing.T) {
	resetViper()
	SetConfigFilePath("/nonexistent/config.yaml")
	cfg := InitConfig()
	assert.Equal(t, DefaultRepositoryDir, cfg.RepositoryDir)
}
