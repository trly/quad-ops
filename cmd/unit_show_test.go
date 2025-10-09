package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/platform"
)

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
	artifacts := []platform.Artifact{
		{
			Path:    "test-service.container",
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
	artifacts := []platform.Artifact{
		{
			Path:    "test-service.container",
			Content: []byte("[Container]\nImage=test\n"),
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
	artifacts := []platform.Artifact{
		{
			Path:    "test-service.container",
			Content: []byte("[Container]\nImage=test\n"),
			Mode:    0644,
			Hash:    "abc123",
		},
		{
			Path:    "test-service.network",
			Content: []byte("[Network]\n"),
			Mode:    0644,
			Hash:    "def456",
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

	err := ExecuteCommand(t, cmd, []string{"test-service"})
	assert.NoError(t, err)
}
