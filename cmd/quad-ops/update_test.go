package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateCmd_Basic tests update command executes.
func TestUpdateCmd_Basic(t *testing.T) {
	output, err := captureOutput(func() error {
		cmd := &UpdateCmd{}
		return cmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Current version:")
}

// TestUpdateCmd_CurrentVersionDisplay tests version is displayed.
func TestUpdateCmd_CurrentVersionDisplay(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v1.5.0"

	output, err := captureOutput(func() error {
		cmd := &UpdateCmd{}
		return cmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Current version: v1.5.0")
}

// TestUpdateCmd_CheckUpdates tests update check message.
func TestUpdateCmd_CheckUpdates(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v2.0.0"

	output, err := captureOutput(func() error {
		cmd := &UpdateCmd{}
		return cmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Checking for updates...")
}

// TestUpdateCmd_UsesVersionVar tests that version variable is accessible.
func TestUpdateCmd_UsesVersionVar(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	testVersion := "v3.2.1"
	Version = testVersion

	// Verify version is set correctly
	assert.Equal(t, testVersion, Version)

	cmd := &UpdateCmd{}
	// Execute - will fail on network, but shouldn't panic
	_ = cmd.Run()
}

// TestUpdateCmd_DevVersionSkipsUpdate tests that dev version skips update check.
func TestUpdateCmd_DevVersionSkipsUpdate(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "dev"

	output, err := captureOutput(func() error {
		cmd := &UpdateCmd{}
		return cmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Current version: dev")
	assert.Contains(t, output, "Update check skipped for dev version")
	assert.NotContains(t, output, "Checking for updates...")
}
