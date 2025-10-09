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
	assert.Contains(t, err.Error(), "failed to list artifacts")
	assert.Contains(t, err.Error(), "storage error")
}

func TestListCommand_Help(t *testing.T) {
	cmd := NewListCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Lists artifacts currently managed by quad-ops")
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

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{Status: false}
	deps := ListDeps{
		CommonDeps:    NewCommonDeps(app.Logger),
		ArtifactStore: artifactStore,
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
		{"systemd container unit", "web-service.container", "web-service"},
		{"systemd with -container suffix", "web-service-container.container", "web-service"},
		{"systemd with -network suffix", "web-service-network.network", "web-service"},
		{"systemd with -volume suffix", "web-service-volume.volume", "web-service"},
		{"systemd with -build suffix", "web-service-build.build", "web-service"},
		{"systemd volume named", "data-volume.volume", "data"},
		{"systemd network named", "app-network-network.network", "app-network"},
		{"systemd build named", "builder-build.build", "builder"},
		{"launchd plist", "com.example.web-service.plist", "web-service"},
		{"launchd simple", "io.quadops.api.plist", "api"},
		{"launchd no prefix", "simple.plist", "simple"},
		{"nested path systemd", "subdir/my-service-container.container", "my-service"},
		{"nested path launchd", "path/to/com.example.svc.plist", "svc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServiceNameFromArtifact(tt.path)
			assert.Equal(t, tt.expected, result)
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
