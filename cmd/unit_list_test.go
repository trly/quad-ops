package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/platform"
)

func TestListCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

func TestListCommand_Success(t *testing.T) {
	artifacts := []platform.Artifact{
		{
			Path:    "web-service.container",
			Content: []byte("[Container]\nImage=nginx\n"),
			Mode:    0644,
			Hash:    "abc123def456",
		},
		{
			Path:    "database.container",
			Content: []byte("[Container]\nImage=postgres\n"),
			Mode:    0644,
			Hash:    "789xyz123abc",
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	assert.NoError(t, err)
}

func TestListCommand_WithStatus(t *testing.T) {
	artifacts := []platform.Artifact{
		{
			Path:    "web-service.container",
			Content: []byte("[Container]\nImage=nginx\n"),
			Mode:    0644,
			Hash:    "abc123def456",
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	lifecycle := &MockLifecycle{
		StatusFunc: func(_ context.Context, _ string) (*platform.ServiceStatus, error) {
			return &platform.ServiceStatus{
				Name:   "web-service",
				Active: true,
				State:  "running",
			}, nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--status"})
	assert.NoError(t, err)
}

func TestListCommand_EmptyList(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{}, nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	assert.NoError(t, err)
}

func TestListCommand_ArtifactStoreError(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return nil, errors.New("storage error")
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list")
	assert.Contains(t, err.Error(), "storage error")
}

func TestListCommand_Help(t *testing.T) {
	cmd := NewListCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Lists deployed artifacts currently managed by quad-ops")
	assert.Contains(t, output, "--status")
}

func TestListCommand_Run(t *testing.T) {
	artifacts := []platform.Artifact{
		{
			Path:    "test-service.container",
			Content: []byte("[Container]\nImage=test\n"),
			Mode:    0644,
			Hash:    "test123hash456",
		},
	}

	repoArtifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{}, nil // Empty filesystem by default
		},
	}

	app := NewAppBuilder(t).
		WithRepoArtifactStore(repoArtifactStore).
		WithArtifactStore(artifactStore).
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{Status: false}
	deps := ListDeps{
		CommonDeps:        NewCommonDeps(app.Logger),
		RepoArtifactStore: repoArtifactStore,
		ArtifactStore:     artifactStore,
	}

	err := listCommand.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

func TestParseServiceNameFromArtifact(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		// Systemd artifacts: base name (without extension) is the unit name
		{"systemd container unit", "web-service.container", "web-service"},
		{"systemd with -container suffix", "web-service-container.container", "web-service-container"},
		{"systemd with -network suffix", "web-service-network.network", "web-service-network"},
		{"systemd with -volume suffix", "web-service-volume.volume", "web-service-volume"},
		{"systemd with -build suffix", "web-service-build.build", "web-service-build"},
		{"systemd volume named", "data-volume.volume", "data-volume"},
		{"systemd network named", "app-network-network.network", "app-network-network"},
		{"systemd build named", "builder-build.build", "builder-build"},

		// Launchd artifacts: extract service name after last dot in label
		{"launchd plist", "com.example.web-service.plist", "web-service"},
		{"launchd simple", "io.quadops.api.plist", "api"},
		{"launchd no prefix", "simple.plist", "simple"},

		// Nested paths
		{"nested path systemd", "subdir/my-service-container.container", "my-service-container"},
		{"nested path launchd", "path/to/com.example.svc.plist", "svc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServiceNameFromArtifact(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseServiceNameFromArtifact_TypeSuffixPreserved tests REGRESSION R5.
// Verifies that systemd artifact names ending with type suffixes preserve those suffixes.
// Expected to FAIL until Step 2 fix is applied.
func TestParseServiceNameFromArtifact_TypeSuffixPreserved(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		note     string
	}{
		{
			name:     "volume literally named 'db-volume'",
			path:     "myapp-db-volume.volume",
			expected: "myapp-db-volume",
			note:     "Should preserve '-volume' in service name",
		},
		{
			name:     "network literally named 'backend-network'",
			path:     "myapp-backend-network.network",
			expected: "myapp-backend-network",
			note:     "Should preserve '-network' in service name",
		},
		{
			name:     "build literally named 'app-build'",
			path:     "myapp-app-build.build",
			expected: "myapp-app-build",
			note:     "Should preserve '-build' in service name",
		},
		{
			name:     "container with -container in actual name",
			path:     "worker-container.container",
			expected: "worker-container",
			note:     "Edge case: service literally named 'worker-container'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServiceNameFromArtifact(tt.path)
			if result != tt.expected {
				t.Logf("REGRESSION R5: %s", tt.note)
				t.Logf("Got '%s', want '%s'", result, tt.expected)
			}
			assert.Equal(t, tt.expected, result, tt.note)
		})
	}
}

func TestExtractArtifactType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"container", "web-service.container", "container"},
		{"volume", "data-volume.volume", "volume"},
		{"network", "app-network.network", "network"},
		{"build", "builder.build", "build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractArtifactType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListCommand_Run_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		artifacts      []platform.Artifact
		opts           ListOptions
		setupLifecycle func() *MockLifecycle
		setupStore     func(artifacts []platform.Artifact) *MockArtifactStore
		expectError    bool
		errorContains  string
	}{
		{
			name: "list artifacts without status",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.container", Hash: "abc123", Mode: 0644},
				{Path: "dev.trly.quad-ops.api.container", Hash: "def456", Mode: 0644},
			},
			opts: ListOptions{Status: false},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			expectError: false,
		},
		{
			name:      "empty artifact list",
			artifacts: []platform.Artifact{},
			opts:      ListOptions{Status: false},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			expectError: false,
		},
		{
			name:      "artifact store error",
			artifacts: nil,
			opts:      ListOptions{Status: false},
			setupStore: func(_ []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return nil, errors.New("database connection failed")
					},
				}
			},
			expectError:   true,
			errorContains: "failed to list",
		},
		{
			name: "list with status - active service",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.container", Hash: "abc123", Mode: 0644},
			},
			opts: ListOptions{Status: true},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			setupLifecycle: func() *MockLifecycle {
				return &MockLifecycle{
					StatusFunc: func(_ context.Context, name string) (*platform.ServiceStatus, error) {
						return &platform.ServiceStatus{
							Name:   name,
							Active: true,
							State:  "running",
						}, nil
					},
				}
			},
			expectError: false,
		},
		{
			name: "list with status - inactive service",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.db.container", Hash: "xyz789", Mode: 0644},
			},
			opts: ListOptions{Status: true},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			setupLifecycle: func() *MockLifecycle {
				return &MockLifecycle{
					StatusFunc: func(_ context.Context, name string) (*platform.ServiceStatus, error) {
						return &platform.ServiceStatus{
							Name:   name,
							Active: false,
							State:  "stopped",
						}, nil
					},
				}
			},
			expectError: false,
		},
		{
			name: "list with status - status error",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.container", Hash: "abc123", Mode: 0644},
			},
			opts: ListOptions{Status: true},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			setupLifecycle: func() *MockLifecycle {
				return &MockLifecycle{
					StatusFunc: func(_ context.Context, _ string) (*platform.ServiceStatus, error) {
						return nil, errors.New("service not found")
					},
				}
			},
			expectError: false,
		},
		{
			name: "filter out non-prefix artifacts",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.container", Hash: "abc123", Mode: 0644},
				{Path: "other-service.container", Hash: "xyz789", Mode: 0644},
			},
			opts: ListOptions{Status: false},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			expectError: false,
		},
		{
			name: "long hash truncation",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.container", Hash: "abcdef1234567890abcdef1234567890", Mode: 0644},
			},
			opts: ListOptions{Status: false},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			expectError: false,
		},
		{
			name: "non-service artifact with status flag",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.config.volume", Hash: "vol123", Mode: 0644},
			},
			opts: ListOptions{Status: true},
			setupStore: func(artifacts []platform.Artifact) *MockArtifactStore {
				return &MockArtifactStore{
					ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
						return artifacts, nil
					},
				}
			},
			setupLifecycle: func() *MockLifecycle {
				return &MockLifecycle{
					StatusFunc: func(_ context.Context, _ string) (*platform.ServiceStatus, error) {
						t.Fatal("StatusFunc should not be called for non-service artifacts")
						return nil, nil
					},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(tt.artifacts)

			// Use the same store for both repo and deployed artifacts
			// (tests use a single mock store that behaves consistently)
			appBuilder := NewAppBuilder(t).
				WithRepoArtifactStore(store).
				WithArtifactStore(store)

			if tt.setupLifecycle != nil {
				lifecycle := tt.setupLifecycle()
				appBuilder = appBuilder.WithLifecycle(lifecycle)
			}

			app := appBuilder.Build(t)

			listCommand := NewListCommand()
			deps := ListDeps{
				CommonDeps:        NewCommonDeps(app.Logger),
				RepoArtifactStore: store,
				ArtifactStore:     store,
			}

			err := listCommand.Run(context.Background(), app, tt.opts, deps)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListCommand_GetLifecycleError(t *testing.T) {
	artifacts := []platform.Artifact{
		{Path: "dev.trly.quad-ops.web.container", Hash: "abc123", Mode: 0644},
	}

	repoArtifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{}, nil
		},
	}

	// Don't set lifecycle - let platform initialization fail naturally
	app := NewAppBuilder(t).
		WithRepoArtifactStore(repoArtifactStore).
		WithArtifactStore(artifactStore).
		WithOS("unsupported-platform").
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{Status: true}
	deps := ListDeps{
		CommonDeps:        NewCommonDeps(app.Logger),
		RepoArtifactStore: repoArtifactStore,
		ArtifactStore:     artifactStore,
	}

	err := listCommand.Run(context.Background(), app, opts, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get lifecycle")
}

func TestListCommand_MixedArtifactTypes(t *testing.T) {
	artifacts := []platform.Artifact{
		{Path: "dev.trly.quad-ops.web.container", Hash: "abc123", Mode: 0644},
		{Path: "dev.trly.quad-ops.data.volume", Hash: "vol456", Mode: 0644},
		{Path: "dev.trly.quad-ops.app-network.network", Hash: "net789", Mode: 0644},
	}

	repoArtifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	lifecycle := &MockLifecycle{
		StatusFunc: func(_ context.Context, name string) (*platform.ServiceStatus, error) {
			return &platform.ServiceStatus{
				Name:   name,
				Active: true,
				State:  "running",
			}, nil
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{}, nil
		},
	}

	app := NewAppBuilder(t).
		WithRepoArtifactStore(repoArtifactStore).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{Status: true}
	deps := ListDeps{
		CommonDeps:        NewCommonDeps(app.Logger),
		RepoArtifactStore: repoArtifactStore,
		ArtifactStore:     artifactStore,
	}

	err := listCommand.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

func TestListCommand_FiltersNonQuadletArtifacts(t *testing.T) {
	artifacts := []platform.Artifact{
		{Path: "web.container", Hash: "abc123", Mode: 0644},
		{Path: ".git/config", Hash: "git123", Mode: 0644},
		{Path: "docker-compose.yml", Hash: "yml456", Mode: 0644},
		{Path: "README.md", Hash: "md789", Mode: 0644},
	}

	repoArtifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{}, nil
		},
	}

	app := NewAppBuilder(t).
		WithRepoArtifactStore(repoArtifactStore).
		WithArtifactStore(artifactStore).
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{Status: false}
	deps := ListDeps{
		CommonDeps:        NewCommonDeps(app.Logger),
		RepoArtifactStore: repoArtifactStore,
		ArtifactStore:     artifactStore,
	}

	err := listCommand.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

// TestListCommand_DefaultUsesDeployedArtifacts tests that unit list defaults to using
// ArtifactStore (deployed directory) instead of RepoArtifactStore (git repo directory).
// This fixes the bug where unit list reported "No deployed artifacts found" even when
// containers were actively running and managed by quad-ops.
// The deployed directory (/etc/containers/systemd/) contains rendered .container, .network, .volume files.
// The repository directory (/var/lib/quad-ops/) contains raw Docker Compose files that don't match platform filters.
func TestListCommand_DefaultUsesDeployedArtifacts(t *testing.T) {
	deployedArtifacts := []platform.Artifact{
		{Path: "infrastructure-reverse-proxy.container", Hash: "c5805a40", Mode: 0644},
		{Path: "infrastructure-unifi-db.container", Hash: "20c60942", Mode: 0644},
		{Path: "infrastructure-unifi-network-application.container", Hash: "a82e7390", Mode: 0644},
	}

	// Track which store was actually called
	var storeUsed string

	// Simulate what the repository directory would return (raw files that don't match platform filters)
	repoArtifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			storeUsed = "repo"
			// Return non-platform artifacts that would be filtered out
			return []platform.Artifact{
				{Path: "quad-ops-deploy/docker-compose.yml", Hash: "comp123", Mode: 0644},
				{Path: "quad-ops-deploy/.git/config", Hash: "gitcfg", Mode: 0644},
			}, nil
		},
	}

	// Simulate what the deployed directory would return (rendered platform-specific files)
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			storeUsed = "deployed"
			return deployedArtifacts, nil
		},
	}

	app := NewAppBuilder(t).
		WithRepoArtifactStore(repoArtifactStore).
		WithArtifactStore(artifactStore).
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{Status: false, UseFilesystem: false} // Default behavior, no flag
	deps := ListDeps{
		CommonDeps:        NewCommonDeps(app.Logger),
		RepoArtifactStore: repoArtifactStore,
		ArtifactStore:     artifactStore,
	}

	// After the fix, the default (without --use-fs-artifacts flag) should use ArtifactStore
	// and find the deployed artifacts
	err := listCommand.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
	// This assertion will fail before the fix is applied
	assert.Equal(t, "deployed", storeUsed, "Expected to use deployed artifact store by default, but used %s", storeUsed)
}
