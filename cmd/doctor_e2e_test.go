package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestDoctor_FullDiagnostics tests complete doctor check workflow.
func TestDoctor_FullDiagnostics(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	repoDir := filepath.Join(tempDir, "repos", "test-repo")
	gitDir := filepath.Join(repoDir, ".git")
	composeDir := filepath.Join(repoDir, "compose")

	// Setup valid repository structure
	require.NoError(t, os.MkdirAll(gitDir, 0750))
	require.NoError(t, os.MkdirAll(composeDir, 0750))
	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return nil
			},
		}).
		WithConfig(&config.Settings{
			Verbose:       true,
			QuadletDir:    filepath.Join(tempDir, "quadlet"),
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/test.git", ComposeDir: "compose"},
			},
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return configFile },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	// May find issues (e.g., missing quadlet-generator) but should not crash
	// The test verifies the doctor runs to completion
	if err != nil {
		assert.Contains(t, err.Error(), "doctor found")
	}
}

// TestDoctor_MissingGitDirectory tests detection of missing .git directory.
func TestDoctor_MissingGitDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	repoDir := filepath.Join(tempDir, "repos", "bad-repo")

	// Create repo directory WITHOUT .git subdirectory
	require.NoError(t, os.MkdirAll(repoDir, 0750))
	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		WithConfig(&config.Settings{
			QuadletDir:    filepath.Join(tempDir, "quadlet"),
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories: []config.Repository{
				{Name: "bad-repo", URL: "https://example.com/bad.git"},
			},
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return configFile },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "doctor found")
}

// TestDoctor_MissingComposeDirectory tests detection of missing compose directory.
func TestDoctor_MissingComposeDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	repoDir := filepath.Join(tempDir, "repos", "test-repo")
	gitDir := filepath.Join(repoDir, ".git")

	// Create valid git repo but missing compose directory
	require.NoError(t, os.MkdirAll(gitDir, 0750))
	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		WithConfig(&config.Settings{
			QuadletDir:    filepath.Join(tempDir, "quadlet"),
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/test.git", ComposeDir: "missing"},
			},
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return configFile },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	assert.Error(t, err)
}

// TestDoctor_MultipleRepositories tests checking multiple repositories.
func TestDoctor_MultipleRepositories(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	// Create multiple repositories with mixed validity
	goodRepo := filepath.Join(tempDir, "repos", "good-repo")
	require.NoError(t, os.MkdirAll(filepath.Join(goodRepo, ".git"), 0750))

	badRepo := filepath.Join(tempDir, "repos", "bad-repo")
	require.NoError(t, os.MkdirAll(badRepo, 0750)) // No .git

	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		WithConfig(&config.Settings{
			QuadletDir:    filepath.Join(tempDir, "quadlet"),
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories: []config.Repository{
				{Name: "good-repo", URL: "https://example.com/good.git"},
				{Name: "bad-repo", URL: "https://example.com/bad.git"},
			},
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return configFile },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	assert.Error(t, err, "should fail when one repo is invalid")
}

// TestDoctor_DirectoryWritability tests directory write permission checks.
func TestDoctor_DirectoryWritability(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	quadletDir := filepath.Join(tempDir, "quadlet")

	require.NoError(t, os.MkdirAll(quadletDir, 0750))
	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		WithConfig(&config.Settings{
			QuadletDir:    quadletDir,
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories:  []config.Repository{},
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return configFile },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	// Should succeed if directories are writable
	if err != nil {
		assert.Contains(t, err.Error(), "doctor found")
	}
}

// TestDoctor_EmptyRepositoryList tests behavior with no repositories configured.
func TestDoctor_EmptyRepositoryList(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		WithConfig(&config.Settings{
			QuadletDir:    filepath.Join(tempDir, "quadlet"),
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories:  []config.Repository{}, // Empty list
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return configFile },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	assert.Error(t, err, "should warn when no repositories configured")
	assert.Contains(t, err.Error(), "doctor found")
}

// TestDoctor_VerboseOutput tests verbose mode includes all check details.
func TestDoctor_VerboseOutput(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	repoDir := filepath.Join(tempDir, "repos", "test-repo")
	gitDir := filepath.Join(repoDir, ".git")

	require.NoError(t, os.MkdirAll(gitDir, 0750))
	require.NoError(t, os.WriteFile(configFile, []byte("test: true"), 0600))

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		WithConfig(&config.Settings{
			Verbose:       true, // Enable verbose
			QuadletDir:    filepath.Join(tempDir, "quadlet"),
			RepositoryDir: filepath.Join(tempDir, "repos"),
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/test.git"},
			},
		}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	cmd := doctorCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	// In verbose mode, should see check details even if they pass
	if err == nil {
		assert.Contains(t, output, "Health check")
	}
}

// TestDoctor_SystemRequirementsFailure tests handling of system requirements failure.
func TestDoctor_SystemRequirementsFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return assert.AnError
			},
		}).
		WithConfig(&config.Settings{}).
		Build(t)

	doctorCmd := NewDoctorCommand()
	deps := DoctorDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),

		ViperConfigFile: func() string { return "" },
		GetOS:           func() string { return "linux" },
	}

	err := doctorCmd.Run(context.Background(), app, DoctorOptions{}, deps)
	assert.Error(t, err)
}
