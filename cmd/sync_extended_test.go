package cmd

import (
	"context"
	"errors"
	"io/fs"
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

// TestSyncCommand_GitSyncFailure tests error handling when git sync fails.
func TestSyncCommand_GitSyncFailure(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, _ []config.Repository) ([]repository.SyncResult, error) {
				return nil, errors.New("git sync failed")
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/repo.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git sync failed")
}

// TestSyncCommand_RepositoryNotFound tests filtering by non-existent repo name.
func TestSyncCommand_RepositoryNotFound(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/repo.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		RepoName: "non-existent-repo",
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository not found: non-existent-repo")
}

// TestSyncCommand_SingleRepoFilter tests syncing a single repository by name.
func TestSyncCommand_SingleRepoFilter(t *testing.T) {
	var syncedRepos []config.Repository

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				syncedRepos = repos
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "repo1", URL: "https://example.com/repo1.git"},
				{Name: "repo2", URL: "https://example.com/repo2.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		RepoName: "repo2",
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
	require.Len(t, syncedRepos, 1)
	assert.Equal(t, "repo2", syncedRepos[0].Name)
}

// TestSyncCommand_MultipleRepositories tests syncing multiple repositories.
func TestSyncCommand_MultipleRepositories(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				results := make([]repository.SyncResult, len(repos))
				for i, repo := range repos {
					results[i] = repository.SyncResult{
						Repository: repo,
						Success:    true,
						Changed:    false,
					}
				}
				return results, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "repo1", URL: "https://example.com/repo1.git"},
				{Name: "repo2", URL: "https://example.com/repo2.git"},
				{Name: "repo3", URL: "https://example.com/repo3.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
}

// TestSyncCommand_RepositorySyncError tests handling of individual repo sync errors.
func TestSyncCommand_RepositorySyncError(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{
						Repository: repos[0],
						Success:    false,
						Error:      errors.New("failed to clone repo"),
					},
				}, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://example.com/repo.git"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

// TestSyncCommand_NoChangesSkipsProcessing tests that unchanged repos are skipped.
func TestSyncCommand_NoChangesSkipsProcessing(t *testing.T) {
	var composeProcessed bool

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
		ComposeProcessor: &MockComposeProcessor{
			ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
				composeProcessed = true
				return []service.Spec{}, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
	assert.False(t, composeProcessed, "Compose should not be processed for unchanged repo")
}

// TestSyncCommand_ForceProcessesUnchangedRepos tests force flag with no compose files triggers reload.
func TestSyncCommand_ForceProcessesUnchangedRepos(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: true},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{
			ReloadFunc: func(_ context.Context) error {
				return nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		Force: true,
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
}

// TestSyncCommand_ReloadOnlyCalledWithChangesOrForce tests reload behavior.
func TestSyncCommand_ReloadOnlyCalledWithChangesOrForce(t *testing.T) {
	var reloadCalled bool

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
		Lifecycle: &MockLifecycle{
			ReloadFunc: func(_ context.Context) error {
				reloadCalled = true
				return nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, err)
	assert.False(t, reloadCalled, "Reload should not be called without changes or force")
}

// TestSyncCommand_RestartChangedServices tests restarting services when anyChanges is true.
func TestSyncCommand_RestartChangedServices(t *testing.T) {
	t.Run("starts services that aren't running", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "test-repo")
		err := os.MkdirAll(repoDir, 0755)
		require.NoError(t, err)

		// Create compose file
		composeContent := `services:
  web:
    image: nginx:latest
`
		err = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)
		require.NoError(t, err)

		var startedServices []string
		var restartedServices []string

		mockLifecycle := &MockLifecycle{
			ReloadFunc: func(_ context.Context) error { return nil },
			StatusFunc: func(_ context.Context, name string) (*platform.ServiceStatus, error) {
				// Service not running
				return &platform.ServiceStatus{Name: name, Active: false}, nil
			},
			StartManyFunc: func(_ context.Context, names []string) map[string]error {
				startedServices = append(startedServices, names...)
				return make(map[string]error)
			},
			RestartManyFunc: func(_ context.Context, names []string) map[string]error {
				restartedServices = append(restartedServices, names...)
				return make(map[string]error)
			},
		}

		mockRenderer := &MockRenderer{
			RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
				return &platform.RenderResult{
					Artifacts: []platform.Artifact{
						{Path: "/tmp/test-web.container", Content: []byte("test")},
					},
					ServiceChanges: map[string]platform.ChangeStatus{
						"test-web": {ArtifactPaths: []string{"/tmp/test-web.container"}},
					},
				}, nil
			},
		}

		deps := SyncDeps{
			CommonDeps: CommonDeps{
				Clock: clock.NewMock(),
				FileSystem: &FileSystemOps{
					MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
				},
				Logger: testutil.NewTestLogger(t),
			},
			GitSyncer: &MockGitSyncer{
				SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
					return []repository.SyncResult{
						{Repository: repos[0], Success: true, Changed: true},
					}, nil
				},
			},
			ComposeProcessor: &MockComposeProcessor{
				ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
					return []service.Spec{
						{Name: "test-web", Container: service.Container{Image: "nginx:latest"}},
					}, nil
				},
			},
			Renderer: mockRenderer,
			ArtifactStore: &MockArtifactStore{
				WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
					return []string{"/tmp/test-web.container"}, nil
				},
			},
			Lifecycle: mockLifecycle,
		}

		app := NewAppBuilder(t).
			WithConfig(&config.Settings{
				RepositoryDir: tmpDir,
				Repositories: []config.Repository{
					{Name: "test-repo"},
				},
			}).
			WithRenderer(mockRenderer).
			WithLifecycle(mockLifecycle).
			Build(t)

		syncCmd := NewSyncCommand()
		opts := SyncOptions{}

		err = syncCmd.Run(context.Background(), app, opts, deps)
		require.NoError(t, err)
		assert.Contains(t, startedServices, "test-web", "Service should be started when not running")
		assert.Empty(t, restartedServices, "Service should not be restarted when not running")
	})

	t.Run("restarts services that are running and changed", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "test-repo")
		err := os.MkdirAll(repoDir, 0755)
		require.NoError(t, err)

		composeContent := `services:
  web:
    image: nginx:latest
`
		err = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)
		require.NoError(t, err)

		var startedServices []string
		var restartedServices []string

		mockLifecycle := &MockLifecycle{
			ReloadFunc: func(_ context.Context) error { return nil },
			StatusFunc: func(_ context.Context, name string) (*platform.ServiceStatus, error) {
				// Service is running
				return &platform.ServiceStatus{Name: name, Active: true}, nil
			},
			StartManyFunc: func(_ context.Context, names []string) map[string]error {
				startedServices = append(startedServices, names...)
				return make(map[string]error)
			},
			RestartManyFunc: func(_ context.Context, names []string) map[string]error {
				restartedServices = append(restartedServices, names...)
				return make(map[string]error)
			},
		}

		mockRenderer := &MockRenderer{
			RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
				return &platform.RenderResult{
					Artifacts: []platform.Artifact{
						{Path: "/tmp/test-web.container", Content: []byte("test")},
					},
					ServiceChanges: map[string]platform.ChangeStatus{
						"test-web": {ArtifactPaths: []string{"/tmp/test-web.container"}},
					},
				}, nil
			},
		}

		deps := SyncDeps{
			CommonDeps: CommonDeps{
				Clock: clock.NewMock(),
				FileSystem: &FileSystemOps{
					MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
				},
				Logger: testutil.NewTestLogger(t),
			},
			GitSyncer: &MockGitSyncer{
				SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
					return []repository.SyncResult{
						{Repository: repos[0], Success: true, Changed: true},
					}, nil
				},
			},
			ComposeProcessor: &MockComposeProcessor{
				ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
					return []service.Spec{
						{Name: "test-web", Container: service.Container{Image: "nginx:latest"}},
					}, nil
				},
			},
			Renderer: mockRenderer,
			ArtifactStore: &MockArtifactStore{
				WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
					return []string{"/tmp/test-web.container"}, nil
				},
			},
			Lifecycle: mockLifecycle,
		}

		app := NewAppBuilder(t).
			WithConfig(&config.Settings{
				RepositoryDir: tmpDir,
				Repositories: []config.Repository{
					{Name: "test-repo"},
				},
			}).
			WithRenderer(mockRenderer).
			WithLifecycle(mockLifecycle).
			Build(t)

		syncCmd := NewSyncCommand()
		opts := SyncOptions{}

		err = syncCmd.Run(context.Background(), app, opts, deps)
		require.NoError(t, err)
		assert.Empty(t, startedServices, "Service should not be started when already running")
		assert.Contains(t, restartedServices, "test-web", "Service should be restarted when running and changed")
	})

	t.Run("handles mixed running and stopped services", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "test-repo")
		err := os.MkdirAll(repoDir, 0755)
		require.NoError(t, err)

		composeContent := `services:
  web:
    image: nginx:latest
  api:
    image: python:latest
`
		err = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)
		require.NoError(t, err)

		var startedServices []string
		var restartedServices []string

		mockLifecycle := &MockLifecycle{
			ReloadFunc: func(_ context.Context) error { return nil },
			StatusFunc: func(_ context.Context, name string) (*platform.ServiceStatus, error) {
				// web is running, api is not
				if name == "test-web" {
					return &platform.ServiceStatus{Name: name, Active: true}, nil
				}
				return &platform.ServiceStatus{Name: name, Active: false}, nil
			},
			StartManyFunc: func(_ context.Context, names []string) map[string]error {
				startedServices = append(startedServices, names...)
				return make(map[string]error)
			},
			RestartManyFunc: func(_ context.Context, names []string) map[string]error {
				restartedServices = append(restartedServices, names...)
				return make(map[string]error)
			},
		}

		mockRenderer := &MockRenderer{
			RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
				return &platform.RenderResult{
					Artifacts: []platform.Artifact{
						{Path: "/tmp/test-web.container", Content: []byte("test")},
						{Path: "/tmp/test-api.container", Content: []byte("test")},
					},
					ServiceChanges: map[string]platform.ChangeStatus{
						"test-web": {ArtifactPaths: []string{"/tmp/test-web.container"}},
						"test-api": {ArtifactPaths: []string{"/tmp/test-api.container"}},
					},
				}, nil
			},
		}

		deps := SyncDeps{
			CommonDeps: CommonDeps{
				Clock: clock.NewMock(),
				FileSystem: &FileSystemOps{
					MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
				},
				Logger: testutil.NewTestLogger(t),
			},
			GitSyncer: &MockGitSyncer{
				SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
					return []repository.SyncResult{
						{Repository: repos[0], Success: true, Changed: true},
					}, nil
				},
			},
			ComposeProcessor: &MockComposeProcessor{
				ProcessFunc: func(_ context.Context, _ *types.Project) ([]service.Spec, error) {
					return []service.Spec{
						{Name: "test-web", Container: service.Container{Image: "nginx:latest"}},
						{Name: "test-api", Container: service.Container{Image: "python:latest"}},
					}, nil
				},
			},
			Renderer: mockRenderer,
			ArtifactStore: &MockArtifactStore{
				WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
					return []string{"/tmp/test-web.container", "/tmp/test-api.container"}, nil
				},
			},
			Lifecycle: mockLifecycle,
		}

		app := NewAppBuilder(t).
			WithConfig(&config.Settings{
				RepositoryDir: tmpDir,
				Repositories: []config.Repository{
					{Name: "test-repo"},
				},
			}).
			WithRenderer(mockRenderer).
			WithLifecycle(mockLifecycle).
			Build(t)

		syncCmd := NewSyncCommand()
		opts := SyncOptions{}

		err = syncCmd.Run(context.Background(), app, opts, deps)
		require.NoError(t, err)
		assert.Contains(t, startedServices, "test-api", "Stopped service should be started")
		assert.Contains(t, restartedServices, "test-web", "Running service should be restarted")
	})
}

// TestSyncCommand_RestartErrors tests handling when no services need restart (logs errors gracefully).
func TestSyncCommand_RestartErrors(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: repos[0], Success: true, Changed: false},
				}, nil
			},
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: t.TempDir(),
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

// TestSyncCommand_TrackChangedServices tests the trackChangedServices method.
func TestSyncCommand_TrackChangedServices(t *testing.T) {
	tests := []struct {
		name             string
		changedPaths     []string
		serviceChanges   map[string]platform.ChangeStatus
		force            bool
		expectedServices []string
	}{
		{
			name:         "service with changed artifact",
			changedPaths: []string{"/path1"},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path1"}},
			},
			force:            false,
			expectedServices: []string{"service1"},
		},
		{
			name:         "service without changed artifact",
			changedPaths: []string{"/path1"},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path2"}},
			},
			force:            false,
			expectedServices: []string{},
		},
		{
			name:         "force restarts all services",
			changedPaths: []string{},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path1"}},
				"service2": {ArtifactPaths: []string{"/path2"}},
			},
			force:            true,
			expectedServices: []string{"service1", "service2"},
		},
		{
			name:         "multiple services with changes",
			changedPaths: []string{"/path1", "/path2"},
			serviceChanges: map[string]platform.ChangeStatus{
				"service1": {ArtifactPaths: []string{"/path1"}},
				"service2": {ArtifactPaths: []string{"/path2"}},
				"service3": {ArtifactPaths: []string{"/path3"}},
			},
			force:            false,
			expectedServices: []string{"service1", "service2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSyncCommand()
			servicesToRestart := make(map[string]bool)

			cmd.trackChangedServices(tt.changedPaths, tt.serviceChanges, tt.force, servicesToRestart)

			assert.Len(t, servicesToRestart, len(tt.expectedServices))
			for _, svc := range tt.expectedServices {
				assert.True(t, servicesToRestart[svc], "Expected %s to be in restart list", svc)
			}
		})
	}
}
