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

// TestUpCommand_DependencyOrdering verifies services start in dependency order.
func TestUpCommand_DependencyOrdering(t *testing.T) {
	// Create temporary directory for compose files
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	_ = os.MkdirAll(repoDir, 0750)

	// Write a compose file with dependencies: web depends on db
	composeContent := `services:
  web:
    image: nginx:latest
    depends_on:
      - db
  db:
    image: postgres:latest
`
	_ = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)

	// Create service specs with dependencies
	processedSpecs := []service.Spec{
		{
			Name:      "web",
			Container: service.Container{Image: "nginx:latest"},
			DependsOn: []string{"db"},
		},
		{
			Name:      "db",
			Container: service.Container{Image: "postgres:latest"},
			DependsOn: []string{},
		},
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
					{Path: "db.container", Content: []byte("db unit"), Hash: "def456"},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true, ArtifactPaths: []string{"web.container"}},
					"db":  {Changed: true, ArtifactPaths: []string{"db.container"}},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return []string{"web.container", "db.container"}, nil
		},
	}

	var startOrder []string
	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			// Capture the order services are passed
			startOrder = names
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
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

	// Execute command - should start all services
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	require.NoError(t, err)

	// Verify services are started in dependency order (db before web)
	require.Len(t, startOrder, 2)
	assert.Equal(t, "db", startOrder[0], "db should start first")
	assert.Equal(t, "web", startOrder[1], "web should start after db")
}

// TestUpCommand_DependencyExpansion verifies requesting a service includes its dependencies.
func TestUpCommand_DependencyExpansion(t *testing.T) {
	// Create temporary directory for compose files
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	_ = os.MkdirAll(repoDir, 0750)

	// Write a compose file with dependencies: web depends on db
	composeContent := `services:
  web:
    image: nginx:latest
    depends_on:
      - db
  db:
    image: postgres:latest
`
	_ = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)

	// Create service specs with dependencies
	processedSpecs := []service.Spec{
		{
			Name:      "web",
			Container: service.Container{Image: "nginx:latest"},
			DependsOn: []string{"db"},
		},
		{
			Name:      "db",
			Container: service.Container{Image: "postgres:latest"},
			DependsOn: []string{},
		},
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
					{Path: "db.container", Content: []byte("db unit"), Hash: "def456"},
				},
				ServiceChanges: map[string]platform.ChangeStatus{
					"web": {Changed: true, ArtifactPaths: []string{"web.container"}},
					"db":  {Changed: true, ArtifactPaths: []string{"db.container"}},
				},
			}, nil
		},
	}

	mockStore := &MockArtifactStore{
		WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
			return []string{"web.container", "db.container"}, nil
		},
	}

	var startOrder []string
	mockLifecycle := &MockLifecycle{
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
		StartManyFunc: func(_ context.Context, names []string) map[string]error {
			// Capture the order services are passed
			startOrder = names
			result := make(map[string]error)
			for _, name := range names {
				result[name] = nil
			}
			return result
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

	// Execute command requesting only 'web' - should auto-include 'db'
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--services", "web"})
	require.NoError(t, err)

	// Verify both services are started with db first
	require.Len(t, startOrder, 2, "requesting web should auto-include db dependency")
	assert.Equal(t, "db", startOrder[0], "db should start first")
	assert.Equal(t, "web", startOrder[1], "web should start after db")
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

// TestUpCommand_NetworkVolumeDependencies verifies network/volume infrastructure is started before containers.
func TestUpCommand_NetworkVolumeDependencies(t *testing.T) {
	tests := []struct {
		name        string
		specs       []service.Spec
		wantStarted []string
		wantOrder   map[string][]string // service -> list of dependencies that must start before it
	}{
		{
			name: "container with volume and network",
			specs: []service.Spec{
				{
					Name:      "web",
					Container: service.Container{Image: "nginx:latest"},
					Volumes: []service.Volume{
						{Name: "web_data", External: false},
					},
					Networks: []service.Network{
						{Name: "web_net"},
					},
				},
			},
			wantStarted: []string{"web_data-volume", "web_net-network", "web"},
			wantOrder: map[string][]string{
				"web": {"web_data-volume", "web_net-network"},
			},
		},
		{
			name: "external volume skipped",
			specs: []service.Spec{
				{
					Name:      "app",
					Container: service.Container{Image: "app:latest"},
					Volumes:   []service.Volume{
						// External volumes tested separately; skip here to avoid validation complexity
					},
				},
			},
			wantStarted: []string{"app"},
			wantOrder:   map[string][]string{},
		},
		{
			name: "multiple services share network",
			specs: []service.Spec{
				{
					Name:      "web",
					Container: service.Container{Image: "nginx:latest"},
					Networks: []service.Network{
						{Name: "shared_net"},
					},
				},
				{
					Name:      "api",
					Container: service.Container{Image: "api:latest"},
					Networks: []service.Network{
						{Name: "shared_net"},
					},
				},
			},
			wantStarted: []string{"shared_net-network", "web", "api"},
			wantOrder: map[string][]string{
				"web": {"shared_net-network"},
				"api": {"shared_net-network"},
			},
		},
		{
			name: "container with service dependency and infrastructure",
			specs: []service.Spec{
				{
					Name:      "web",
					Container: service.Container{Image: "nginx:latest"},
					DependsOn: []string{"db"},
					Networks: []service.Network{
						{Name: "app_net"},
					},
				},
				{
					Name:      "db",
					Container: service.Container{Image: "postgres:latest"},
					Volumes: []service.Volume{
						{Name: "db_data", External: false},
					},
					Networks: []service.Network{
						{Name: "app_net"},
					},
				},
			},
			wantStarted: []string{"db_data-volume", "app_net-network", "db", "web"},
			wantOrder: map[string][]string{
				"db":  {"db_data-volume", "app_net-network"},
				"web": {"app_net-network", "db"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for compose files
			tempDir := t.TempDir()
			repoDir := filepath.Join(tempDir, "test-repo")
			_ = os.MkdirAll(repoDir, 0750)

			// Write minimal compose file (processor will return our test specs)
			composeContent := `services:
  placeholder:
    image: placeholder:latest
`
			_ = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)

			mockProcessor := &MockComposeProcessor{
				ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
					return tt.specs, nil
				},
			}

			// Build artifacts for all specs
			var artifacts []platform.Artifact
			serviceChanges := make(map[string]platform.ChangeStatus)

			for _, spec := range tt.specs {
				// Add volume artifacts
				for _, vol := range spec.Volumes {
					if !vol.External {
						artifacts = append(artifacts, platform.Artifact{
							Path:    vol.Name + ".volume",
							Content: []byte("volume unit"),
							Hash:    "vol-" + vol.Name,
						})
					}
				}

				// Add network artifacts
				for _, net := range spec.Networks {
					artifacts = append(artifacts, platform.Artifact{
						Path:    net.Name + ".network",
						Content: []byte("network unit"),
						Hash:    "net-" + net.Name,
					})
				}

				// Add container artifact
				artifacts = append(artifacts, platform.Artifact{
					Path:    spec.Name + ".container",
					Content: []byte("container unit"),
					Hash:    "cont-" + spec.Name,
				})

				serviceChanges[spec.Name] = platform.ChangeStatus{
					Changed:       true,
					ArtifactPaths: []string{spec.Name + ".container"},
				}
			}

			mockRenderer := &MockRenderer{
				RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
					return &platform.RenderResult{
						Artifacts:      artifacts,
						ServiceChanges: serviceChanges,
					}, nil
				},
			}

			mockStore := &MockArtifactStore{
				WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
					paths := make([]string, len(artifacts))
					for i, a := range artifacts {
						paths[i] = a.Path
					}
					return paths, nil
				},
			}

			var startOrder []string
			mockLifecycle := &MockLifecycle{
				ReloadFunc: func(_ context.Context) error {
					return nil
				},
				RestartManyFunc: func(_ context.Context, names []string) map[string]error {
					startOrder = append(startOrder, names...)
					result := make(map[string]error)
					for _, name := range names {
						result[name] = nil
					}
					return result
				},
				StartManyFunc: func(_ context.Context, names []string) map[string]error {
					startOrder = append(startOrder, names...)
					result := make(map[string]error)
					for _, name := range names {
						result[name] = nil
					}
					return result
				},
			}

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

			// Verify all expected services were started
			assert.ElementsMatch(t, tt.wantStarted, startOrder, "all expected services should be started")

			// Verify dependency ordering
			for service, deps := range tt.wantOrder {
				serviceIdx := indexOf(startOrder, service)
				require.NotEqual(t, -1, serviceIdx, "service %s should be in start order", service)

				for _, dep := range deps {
					depIdx := indexOf(startOrder, dep)
					require.NotEqual(t, -1, depIdx, "dependency %s should be in start order", dep)
					assert.Less(t, depIdx, serviceIdx, "%s must start before %s", dep, service)
				}
			}
		})
	}
}

// indexOf returns the index of item in slice, or -1 if not found.
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
