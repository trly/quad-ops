package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestWorkflow_SyncUpDown tests the complete workflow: sync → up → down.
func TestWorkflow_SyncUpDown(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repos", "test-app")
	quadletDir := filepath.Join(tempDir, "quadlet")

	// Create repository structure with compose file
	require.NoError(t, os.MkdirAll(repoDir, 0750))
	composeContent := `services:
  web:
    image: nginx:latest
  db:
    image: postgres:latest
`
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600))

	// Track service lifecycle
	var startedServices []string
	var stoppedServices []string

	// Mock components
	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return []service.Spec{
				{Name: "web", Container: service.Container{Image: "nginx:latest"}},
				{Name: "db", Container: service.Container{Image: "postgres:latest"}},
			}, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web unit")},
					{Path: "db.container", Content: []byte("db unit")},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true},
					"db":  {Changed: true},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return []string{"web.container", "db.container"}, nil
		},
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "web.container"},
				{Path: "db.container"},
			}, nil
		},
		DeleteFunc: func(_ context.Context, _ []string) error {
			return nil
		},
	}

	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			startedServices = append(startedServices, names...)
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
		},
		StopManyFunc: func(_ context.Context, names []string) map[string]error {
			stoppedServices = append(stoppedServices, names...)
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
		},
	}

	mockGitSyncer := &MockGitSyncer{
		SyncAllFunc: func(_ context.Context, _ []config.Repository) ([]repository.SyncResult, error) {
			return []repository.SyncResult{
				{Repository: config.Repository{Name: "test-app"}, Success: true, Changed: true},
			}, nil
		},
	}

	cfg := &config.Settings{
		RepositoryDir: filepath.Join(tempDir, "repos"),
		QuadletDir:    quadletDir,
		Repositories: []config.Repository{
			{Name: "test-app", URL: "https://example.com/test-app.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	// Step 1: Sync repositories
	syncCmd := NewSyncCommand()
	syncDeps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer:        mockGitSyncer,
		ComposeProcessor: mockProcessor,
		Renderer:         mockRenderer,
		ArtifactStore:    mockStore,
		Lifecycle:        mockLifecycle,
	}

	err := syncCmd.Run(context.Background(), app, SyncOptions{}, syncDeps)
	require.NoError(t, err, "sync should succeed")

	// Step 2: Bring services up
	upCmd := NewUpCommand()
	upCmdCobra := upCmd.GetCobraCommand()
	SetupCommandContext(upCmdCobra, app)

	err = ExecuteCommand(t, upCmdCobra, []string{})
	require.NoError(t, err, "up should succeed")

	assert.Len(t, startedServices, 2, "should start 2 services")
	assert.Contains(t, startedServices, "web")
	assert.Contains(t, startedServices, "db")

	// Step 3: Bring services down
	downCmd := NewDownCommand()
	downCmdCobra := downCmd.GetCobraCommand()
	SetupCommandContext(downCmdCobra, app)

	err = ExecuteCommand(t, downCmdCobra, []string{})
	require.NoError(t, err, "down should succeed")

	assert.Len(t, stoppedServices, 2, "should stop 2 services")
	assert.Contains(t, stoppedServices, "web")
	assert.Contains(t, stoppedServices, "db")
}

// TestWorkflow_UpWithDependencies tests starting services with dependencies.
func TestWorkflow_UpWithDependencies(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repos", "app-with-deps")
	require.NoError(t, os.MkdirAll(repoDir, 0750))

	composeContent := `services:
  web:
    image: nginx:latest
    depends_on:
      - api
  api:
    image: httpd:latest
    depends_on:
      - db
  db:
    image: postgres:latest
`
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600))

	var startOrder []string

	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return []service.Spec{
				{Name: "web", Container: service.Container{Image: "nginx:latest"}, DependsOn: []string{"api"}},
				{Name: "api", Container: service.Container{Image: "httpd:latest"}, DependsOn: []string{"db"}},
				{Name: "db", Container: service.Container{Image: "postgres:latest"}},
			}, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web")},
					{Path: "api.container", Content: []byte("api")},
					{Path: "db.container", Content: []byte("db")},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true},
					"api": {Changed: true},
					"db":  {Changed: true},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return []string{"web.container", "api.container", "db.container"}, nil
		},
	}

	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			startOrder = names
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
		},
	}

	cfg := &config.Settings{
		RepositoryDir: filepath.Join(tempDir, "repos"),
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
		Repositories: []config.Repository{
			{Name: "app-with-deps", URL: "https://example.com/app.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	require.NoError(t, err)

	// Verify dependency order: db → api → web
	require.Len(t, startOrder, 3)
	assert.Equal(t, "db", startOrder[0])
	assert.Equal(t, "api", startOrder[1])
	assert.Equal(t, "web", startOrder[2])
}

// TestWorkflow_SelectiveOperations tests operating on specific services.
func TestWorkflow_SelectiveOperations(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repos", "multi-service")
	require.NoError(t, os.MkdirAll(repoDir, 0750))

	composeContent := `services:
  web:
    image: nginx:latest
  api:
    image: httpd:latest
  db:
    image: postgres:latest
`
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600))

	var startedServices []string
	var stoppedServices []string

	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return []service.Spec{
				{Name: "web", Container: service.Container{Image: "nginx:latest"}},
				{Name: "api", Container: service.Container{Image: "httpd:latest"}},
				{Name: "db", Container: service.Container{Image: "postgres:latest"}},
			}, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web")},
					{Path: "api.container", Content: []byte("api")},
					{Path: "db.container", Content: []byte("db")},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true},
					"api": {Changed: true},
					"db":  {Changed: true},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return []string{"web.container", "api.container", "db.container"}, nil
		},
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "web.container"},
				{Path: "api.container"},
				{Path: "db.container"},
			}, nil
		},
	}

	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			startedServices = append(startedServices, names...)
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
		},
		StopManyFunc: func(_ context.Context, names []string) map[string]error {
			stoppedServices = append(stoppedServices, names...)
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
		},
	}

	cfg := &config.Settings{
		RepositoryDir: filepath.Join(tempDir, "repos"),
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
		Repositories: []config.Repository{
			{Name: "multi-service", URL: "https://example.com/app.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	// Start only web and api
	upCmd := NewUpCommand()
	upCmdCobra := upCmd.GetCobraCommand()
	SetupCommandContext(upCmdCobra, app)

	err := ExecuteCommand(t, upCmdCobra, []string{"--services", "web,api"})
	require.NoError(t, err)

	assert.Len(t, startedServices, 2)
	assert.Contains(t, startedServices, "web")
	assert.Contains(t, startedServices, "api")
	assert.NotContains(t, startedServices, "db")

	// Stop only web
	downCmd := NewDownCommand()
	downCmdCobra := downCmd.GetCobraCommand()
	SetupCommandContext(downCmdCobra, app)

	err = ExecuteCommand(t, downCmdCobra, []string{"--services", "web"})
	require.NoError(t, err)

	assert.Len(t, stoppedServices, 1)
	assert.Contains(t, stoppedServices, "web")
}

// TestWorkflow_DryRunMode tests dry-run across commands.
func TestWorkflow_DryRunMode(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repos", "test")
	require.NoError(t, os.MkdirAll(repoDir, 0750))

	composeContent := `services:
  web:
    image: nginx:latest
`
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600))

	writeCalled := false
	startCalled := false

	mockProcessor := &MockComposeProcessor{
		ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
			return []service.Spec{
				{Name: "web", Container: service.Container{Image: "nginx:latest"}},
			}, nil
		},
	}

	mockRenderer := &MockRenderer{
		RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
			return &platform.RenderResult{
				Artifacts: []platform.Artifact{
					{Path: "web.container", Content: []byte("web")},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			writeCalled = true
			return nil, nil
		},
	}

	mockLifecycle := &MockLifecycle{
		StartManyFunc: func(_ context.Context, _ []string) map[string]error {
			startCalled = true
			return nil
		},
	}

	mockGitSyncer := &MockGitSyncer{
		SyncAllFunc: func(_ context.Context, _ []config.Repository) ([]repository.SyncResult, error) {
			return []repository.SyncResult{
				{Repository: config.Repository{Name: "test"}, Success: true, Changed: true},
			}, nil
		},
	}

	cfg := &config.Settings{
		RepositoryDir: filepath.Join(tempDir, "repos"),
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
		Repositories: []config.Repository{
			{Name: "test", URL: "https://example.com/test.git"},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(cfg).
		WithComposeProcessor(mockProcessor).
		WithRenderer(mockRenderer).
		WithArtifactStore(mockStore).
		WithLifecycle(mockLifecycle).
		Build(t)

	// Dry-run sync
	syncCmd := NewSyncCommand()
	syncDeps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer:        mockGitSyncer,
		ComposeProcessor: mockProcessor,
		Renderer:         mockRenderer,
		ArtifactStore:    mockStore,
		Lifecycle:        mockLifecycle,
	}

	err := syncCmd.Run(context.Background(), app, SyncOptions{DryRun: true}, syncDeps)
	require.NoError(t, err)

	// Dry-run up
	upCmd := NewUpCommand()
	upCmdCobra := upCmd.GetCobraCommand()
	SetupCommandContext(upCmdCobra, app)

	err = ExecuteCommand(t, upCmdCobra, []string{"--dry-run"})
	require.NoError(t, err)

	assert.False(t, writeCalled, "dry-run should not write artifacts")
	assert.False(t, startCalled, "dry-run should not start services")
}
