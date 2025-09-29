package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/repository"
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

func TestListCommand_InvalidUnitType(t *testing.T) {
	app := NewAppBuilder(t).Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--type", "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid unit type")
}

func TestListCommand_Success(t *testing.T) {
	units := []repository.Unit{
		{
			ID:        1,
			Name:      "web-service",
			Type:      "container",
			SHA1Hash:  []byte{0x12, 0x34, 0x56},
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			Name:      "database",
			Type:      "container",
			SHA1Hash:  []byte{0x78, 0x9a, 0xbc},
			UpdatedAt: time.Now(),
		},
	}

	unitRepo := &MockUnitRepo{
		FindByUnitTypeFunc: func(_ string) ([]repository.Unit, error) {
			return units, nil
		},
	}

	unitManager := &MockUnitManager{}

	app := NewAppBuilder(t).
		WithUnitRepo(unitRepo).
		WithUnitManager(unitManager).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	assert.NoError(t, err)
}

func TestListCommand_AllUnits(t *testing.T) {
	units := []repository.Unit{
		{
			ID:        1,
			Name:      "web-service",
			Type:      "container",
			SHA1Hash:  []byte{0x12, 0x34, 0x56},
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			Name:      "data-volume",
			Type:      "volume",
			SHA1Hash:  []byte{0x78, 0x9a, 0xbc},
			UpdatedAt: time.Now(),
		},
	}

	unitRepo := &MockUnitRepo{
		FindAllFunc: func() ([]repository.Unit, error) {
			return units, nil
		},
	}

	unitManager := &MockUnitManager{}

	app := NewAppBuilder(t).
		WithUnitRepo(unitRepo).
		WithUnitManager(unitManager).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--type", "all"})
	assert.NoError(t, err)
}

func TestListCommand_RepositoryError(t *testing.T) {
	unitRepo := &MockUnitRepo{
		FindByUnitTypeFunc: func(_ string) ([]repository.Unit, error) {
			return nil, errors.New("database connection failed")
		},
	}

	app := NewAppBuilder(t).
		WithUnitRepo(unitRepo).
		Build(t)

	cmd := NewListCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error finding units")
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestListCommand_Help(t *testing.T) {
	cmd := NewListCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Lists units currently managed by quad-ops")
	assert.Contains(t, output, "--type")
}

func TestListCommand_Run(t *testing.T) {
	units := []repository.Unit{
		{
			ID:        1,
			Name:      "test-unit",
			Type:      "container",
			SHA1Hash:  []byte{0x12, 0x34, 0x56},
			UpdatedAt: time.Now(),
		},
	}

	unitRepo := &MockUnitRepo{
		FindByUnitTypeFunc: func(_ string) ([]repository.Unit, error) {
			return units, nil
		},
	}

	unitManager := &MockUnitManager{}

	app := NewAppBuilder(t).
		WithUnitRepo(unitRepo).
		WithUnitManager(unitManager).
		Build(t)

	listCommand := NewListCommand()
	opts := ListOptions{UnitType: "container"}
	deps := ListDeps{CommonDeps: NewCommonDeps(app.Logger)}

	err := listCommand.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
}

func TestValidateUnitType(t *testing.T) {
	tests := []struct {
		name      string
		unitType  string
		wantError bool
	}{
		{"valid container", "container", false},
		{"valid volume", "volume", false},
		{"valid network", "network", false},
		{"valid image", "image", false},
		{"valid all", "all", false},
		{"invalid type", "invalid", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUnitType(tt.unitType)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
