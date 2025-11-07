package cmd

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/platform/launchd"
)

// testServiceArtifact creates a platform-appropriate service artifact for testing.
// On Linux, it returns a .container file. On macOS, it returns a .plist with the proper label prefix.
func testServiceArtifact(t *testing.T, app *App, name, content, hash string) platform.Artifact {
	t.Helper()
	if runtime.GOOS == "darwin" {
		opts := launchd.OptionsFromSettings(app.Config.RepositoryDir, app.Config.QuadletDir, app.Config.UserMode)
		return platform.Artifact{
			Path:    fmt.Sprintf("%s.%s.plist", opts.LabelPrefix, name),
			Content: []byte(content),
			Mode:    0644,
			Hash:    hash,
		}
	}
	return platform.Artifact{
		Path:    fmt.Sprintf("%s.container", name),
		Content: []byte(content),
		Mode:    0644,
		Hash:    hash,
	}
}

func TestShowCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

func TestShowCommand_Success(t *testing.T) {
	app := NewAppBuilder(t).Build(t)

	artifacts := []platform.Artifact{
		testServiceArtifact(t, app, "test-service", "[Container]\nImage=nginx\n", "abc123"),
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	app.ArtifactStore = artifactStore

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"test-service"})
	assert.NoError(t, err)
}

func TestShowCommand_ServiceNotFound(t *testing.T) {
	artifacts := []platform.Artifact{
		{
			Path:    "other-service.container",
			Content: []byte("[Container]\nImage=nginx\n"),
			Mode:    0644,
			Hash:    "abc123",
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

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"missing-service"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no artifact found")
}

func TestShowCommand_ArtifactStoreError(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return nil, errors.New("storage error")
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		Build(t)

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"test-service"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list artifacts")
	assert.Contains(t, err.Error(), "storage error")
}

func TestShowCommand_Help(t *testing.T) {
	cmd := NewShowCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Show the contents of a service artifact")
}

func TestShowCommand_Run(t *testing.T) {
	app := NewAppBuilder(t).Build(t)

	artifacts := []platform.Artifact{
		testServiceArtifact(t, app, "test-service", "[Container]\nImage=test\n", "abc123"),
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	app.ArtifactStore = artifactStore

	showCommand := NewShowCommand()
	opts := ShowOptions{}
	deps := ShowDeps{
		CommonDeps:    NewCommonDeps(app.Logger),
		ArtifactStore: artifactStore,
	}

	err := showCommand.Run(context.Background(), app, opts, deps, "test-service")
	assert.NoError(t, err)
}

func TestShowCommand_MultipleArtifacts(t *testing.T) {
	app := NewAppBuilder(t).Build(t)

	artifacts := []platform.Artifact{
		testServiceArtifact(t, app, "test-service", "[Container]\nImage=test\n", "abc123"),
	}

	// Add a network artifact (non-service, should be filtered out on macOS but kept on Linux)
	if runtime.GOOS == "linux" {
		artifacts = append(artifacts, platform.Artifact{
			Path:    "test-service.network",
			Content: []byte("[Network]\n"),
			Mode:    0644,
			Hash:    "def456",
		})
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	app.ArtifactStore = artifactStore

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"test-service"})
	assert.NoError(t, err)
}

func TestShowCommand_FiltersNonQuadletArtifacts(t *testing.T) {
	app := NewAppBuilder(t).Build(t)

	artifacts := []platform.Artifact{
		testServiceArtifact(t, app, "web", "[Container]\nImage=nginx\n", "abc123"),
		{
			Path:    ".git/config",
			Content: []byte("git config"),
			Mode:    0644,
			Hash:    "git123",
		},
		{
			Path:    "docker-compose.yml",
			Content: []byte("version: 3"),
			Mode:    0644,
			Hash:    "yml456",
		},
	}

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return artifacts, nil
		},
	}

	app.ArtifactStore = artifactStore

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"web"})
	assert.NoError(t, err)
}
