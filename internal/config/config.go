// Package config provides application configuration structures and utilities.
package config

import (
	"os"
	"path/filepath"
)

// getuid is the function used to retrieve the current user ID.
// It is a variable to allow tests to simulate root/non-root environments.
var getuid = os.Getuid

// AppConfig represents the application configuration loaded from a YAML file.
type AppConfig struct {
	RepositoryDir string `yaml:"repositoryDir,omitempty"`
	QuadletDir    string `yaml:"quadletDir,omitempty"`
	Repositories  []struct {
		Name       string `yaml:"name"`
		URL        string `yaml:"url"`
		Ref        string `yaml:"ref,omitempty"`
		ComposeDir string `yaml:"composeDir,omitempty"`
	} `yaml:"repositories"`
}

// IsUserMode returns true if running as non-root user (uid != 0).
func IsUserMode() bool {
	return getuid() != 0
}

// GetRepositoryDir returns the repository directory, using the default based on user mode if not configured.
func (c *AppConfig) GetRepositoryDir() string {
	if c.RepositoryDir != "" {
		return c.RepositoryDir
	}
	if IsUserMode() {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local/share/quad-ops")
	}
	return "/var/lib/quad-ops"
}

// GetStateFilePath returns the path to the deployment state file.
func (c *AppConfig) GetStateFilePath() string {
	if IsUserMode() {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config/quad-ops/state.json")
	}
	return "/var/lib/quad-ops/state.json"
}

// GetQuadletDir returns the quadlet directory, using the default based on user mode if not configured.
func (c *AppConfig) GetQuadletDir() string {
	if c.QuadletDir != "" {
		return c.QuadletDir
	}
	if IsUserMode() {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config/containers/systemd")
	}
	return "/etc/containers/systemd"
}
