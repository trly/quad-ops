package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trly/quad-ops/internal/config"
)

func TestNewRepository(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize config before running the test
	cfg := &config.Config{
		RepositoryDir: tmpDir,
		Verbose:       true,
	}
	config.SetConfig(cfg)

	testRepo := config.Repository{
		Name:   "test-repo",
		URL:    "https://github.com/test/repo.git",
		Target: "main",
	}

	repo := NewRepository(testRepo)

	if repo.URL != testRepo.URL {
		t.Errorf("Expected URL %s, got %s", testRepo.URL, repo.URL)
	}

	expectedPath := filepath.Join(tmpDir, testRepo.Name)
	if repo.Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, repo.Path)
	}

	if repo.Target != testRepo.Target {
		t.Errorf("Expected target %s, got %s", testRepo.Target, repo.Target)
	}
}

func TestSyncRepository(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up test config
	cfg := &config.Config{
		RepositoryDir: tmpDir,
		Verbose:       true,
	}
	config.SetConfig(cfg)

	testRepo := config.Repository{
		Name:   "test-repo",
		URL:    "https://github.com/test/repo.git",
		Target: "main",
	}

	repo := NewRepository(testRepo)

	// Test invalid repository URL
	err = repo.SyncRepository()
	if err == nil {
		t.Error("Expected error for invalid repository URL")
	}
}

func TestCheckoutTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		RepositoryDir: tmpDir,
		Verbose:       true,
	}
	config.SetConfig(cfg)

	testRepo := config.Repository{
		Name:   "test-repo",
		URL:    "https://github.com/test/repo.git",
		Target: "main",
	}

	repo := NewRepository(testRepo)
	if repo.Target != "main" {
		t.Errorf("Expected target main, got %s", repo.Target)
	}
}
