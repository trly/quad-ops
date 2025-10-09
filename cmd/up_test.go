package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/service"
)

// TestUpCommand_ValidationFailure verifies that validation failures are handled correctly.
func TestUpCommand_ValidationFailure(t *testing.T) {
	// Create app with failing validator
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	// Setup command with app in context
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute PreRunE (which should trigger validation)
	err := cmd.PreRunE(cmd, []string{})

	// Verify error was returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

// TestUpCommand_Success verifies successful service startup with compose processing.
func TestUpCommand_Success(t *testing.T) {
	// Create temporary directory for compose files
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	_ = os.MkdirAll(repoDir, 0750)

	// Write a simple compose file
	composeContent := `services:
  web:
    image: nginx:latest
  api:
    image: httpd:latest
`
	_ = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)

	// Create mocks
	processedSpecs := []service.Spec{
		{Name: "web", Container: service.Container{Image: "nginx:latest"}},
		{Name: "api", Container: service.Container{Image: "httpd:latest"}},
	}

	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return processedSpecs, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web unit"), Hash: "abc123"},
					{Path: "api.container", Content: []byte("api unit"), Hash: "def456"},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true, ArtifactPaths: []string{"web.container"}},
					"api": {Changed: true, ArtifactPaths: []string{"api.container"}},
				},
			}, nil
		},
	}

	changedPaths := []string{"web.container", "api.container"}
	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return changedPaths, nil
		},
	}

	startCalls := make(map[string]error)
	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			for _, name := range names {
				startCalls[name] = nil
			}
			return startCalls
		},
	}

	// Build app with test repositories
	cfg := &config.Settings{
		RepositoryDir: tempDir,
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
		Repositories: []config.Repository{
			{Name: "test-repo", URL: "https://example.com/test.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	// Execute command
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	require.NoError(t, err)

	// Verify services were started
	assert.Len(t, startCalls, 2)
	assert.Contains(t, startCalls, "web")
	assert.Contains(t, startCalls, "api")
}

// TestUpCommand_DryRun verifies dry-run mode behavior.
func TestUpCommand_DryRun(t *testing.T) {
	// Create temporary directory for compose files
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	_ = os.MkdirAll(repoDir, 0750)

	// Write a simple compose file
	composeContent := `services:
  web:
    image: nginx:latest
`
	_ = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)

	// Create mocks
	processedSpecs := []service.Spec{
		{Name: "web", Container: service.Container{Image: "nginx:latest"}},
	}

	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return processedSpecs, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web unit"), Hash: "abc123"},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true, ArtifactPaths: []string{"web.container"}},
				},
			}, nil
		},
	}

	writeCalled := false
	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			writeCalled = true
			return nil, nil
		},
	}

	startCalled := false
	mockLifecycle := &MockLifecycle{
		StartManyFunc: func(_ context.Context, _ []string) map[string]error {
			startCalled = true
			return nil
		},
	}

	// Build app
	cfg := &config.Settings{
		RepositoryDir: tempDir,
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
		Repositories: []config.Repository{
			{Name: "test-repo", URL: "https://example.com/test.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	// Execute command with dry-run flag
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--dry-run"})
	require.NoError(t, err)

	// Verify no actual changes were made
	assert.False(t, writeCalled, "Write should not be called in dry-run")
	assert.False(t, startCalled, "StartMany should not be called in dry-run")
}

// TestUpCommand_FilterByServices verifies filtering by --services flag.
func TestUpCommand_FilterByServices(t *testing.T) {
	// Create temporary directory for compose files
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	_ = os.MkdirAll(repoDir, 0750)

	// Write a compose file with multiple services
	composeContent := `services:
  web:
    image: nginx:latest
  api:
    image: httpd:latest
  db:
    image: postgres:latest
`
	_ = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)

	// Create mocks
	processedSpecs := []service.Spec{
		{Name: "web", Container: service.Container{Image: "nginx:latest"}},
		{Name: "api", Container: service.Container{Image: "httpd:latest"}},
		{Name: "db", Container: service.Container{Image: "postgres:latest"}},
	}

	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return processedSpecs, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web unit"), Hash: "abc123"},
					{Path: "api.container", Content: []byte("api unit"), Hash: "def456"},
					{Path: "db.container", Content: []byte("db unit"), Hash: "ghi789"},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true, ArtifactPaths: []string{"web.container"}},
					"api": {Changed: true, ArtifactPaths: []string{"api.container"}},
					"db":  {Changed: true, ArtifactPaths: []string{"db.container"}},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return []string{"web.container", "api.container", "db.container"}, nil
		},
	}

	startCalls := make(map[string]error)
	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			for _, name := range names {
				startCalls[name] = nil
			}
			return startCalls
		},
	}

	// Build app
	cfg := &config.Settings{
		RepositoryDir: tempDir,
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
		Repositories: []config.Repository{
			{Name: "test-repo", URL: "https://example.com/test.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	// Execute command with --services filter
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--services", "web,api"})
	require.NoError(t, err)

	// Verify only specified services were started
	assert.Len(t, startCalls, 2)
	assert.Contains(t, startCalls, "web")
	assert.Contains(t, startCalls, "api")
	assert.NotContains(t, startCalls, "db")
}

// TestUpCommand_Help verifies help output.
func TestUpCommand_Help(t *testing.T) {
	cmd := NewUpCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Process Docker Compose files")
	assert.Contains(t, output, "--services")
	assert.Contains(t, output, "--dry-run")
	assert.Contains(t, output, "--repo")
	assert.Contains(t, output, "--force")
}
