package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func fakeRoot(t *testing.T) {
	t.Helper()
	orig := getuid
	getuid = func() int { return 0 }
	t.Cleanup(func() { getuid = orig })
}

func TestIsUserMode(t *testing.T) {
	assert.True(t, IsUserMode())
}

func TestIsUserMode_Root(t *testing.T) {
	fakeRoot(t)
	assert.False(t, IsUserMode())
}

func TestGetRepositoryDir_Configured(t *testing.T) {
	cfg := &AppConfig{RepositoryDir: "/custom/repo/dir"}
	assert.Equal(t, "/custom/repo/dir", cfg.GetRepositoryDir())
}

func TestGetRepositoryDir_DefaultUserMode(t *testing.T) {
	cfg := &AppConfig{}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local/share/quad-ops")
	assert.Equal(t, expected, cfg.GetRepositoryDir())
}

func TestGetStateFilePath_DefaultUserMode(t *testing.T) {
	cfg := &AppConfig{}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config/quad-ops/state.json")
	assert.Equal(t, expected, cfg.GetStateFilePath())
}

func TestGetQuadletDir_Configured(t *testing.T) {
	cfg := &AppConfig{QuadletDir: "/custom/quadlet/dir"}
	assert.Equal(t, "/custom/quadlet/dir", cfg.GetQuadletDir())
}

func TestGetQuadletDir_DefaultUserMode(t *testing.T) {
	cfg := &AppConfig{}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config/containers/systemd")
	assert.Equal(t, expected, cfg.GetQuadletDir())
}

func TestGetRepositoryDir_DefaultSystemMode(t *testing.T) {
	fakeRoot(t)
	cfg := &AppConfig{}
	assert.Equal(t, "/var/lib/quad-ops", cfg.GetRepositoryDir())
}

func TestGetStateFilePath_DefaultSystemMode(t *testing.T) {
	fakeRoot(t)
	cfg := &AppConfig{}
	assert.Equal(t, "/var/lib/quad-ops/state.json", cfg.GetStateFilePath())
}

func TestGetQuadletDir_DefaultSystemMode(t *testing.T) {
	fakeRoot(t)
	cfg := &AppConfig{}
	assert.Equal(t, "/etc/containers/systemd", cfg.GetQuadletDir())
}
