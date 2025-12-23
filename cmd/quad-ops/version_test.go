package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput captures stdout during command execution.
func captureOutput(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	err = fn()

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	_ = w.Close()
	os.Stdout = old
	output := <-outC

	return output, err
}

// TestVersionCommand_Basic tests version command.
func TestVersionCommand_Basic(t *testing.T) {
	output, err := captureOutput(func() error {
		cmd := &VersionCmd{}
		return cmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops")
}

// TestVersionCommand_OutputContainsVersionInfo tests version output details.
func TestVersionCommand_OutputContainsVersionInfo(t *testing.T) {
	output, err := captureOutput(func() error {
		cmd := &VersionCmd{}
		return cmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops version")
	assert.Contains(t, output, "commit:")
	assert.Contains(t, output, "built:")
	assert.Contains(t, output, "go:")
}

// TestVersionCommand_Help tests help output through kong.
func TestVersionCommand_Help(t *testing.T) {
	// Verify version command is registered in CLI
	cli := CLI{}
	assert.Equal(t, &VersionCmd{}, &cli.Version)
}

// TestVersionCommand_DevVersion tests development version handling.
func TestVersionCommand_DevVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "dev"

	output, err := captureOutput(func() error {
		vcmd := &VersionCmd{}
		return vcmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops version dev")
	assert.Contains(t, output, "Skipping update check for development build")
}

// TestVersionCommand_ReleaseVersion tests release version handling.
func TestVersionCommand_ReleaseVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v1.0.0"

	output, err := captureOutput(func() error {
		vcmd := &VersionCmd{}
		return vcmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops version v1.0.0")
	assert.Contains(t, output, "Checking for updates")
}

// TestVersionCommand_BuildInfo tests build information display.
func TestVersionCommand_BuildInfo(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	defer func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	}()

	Version = "v1.2.3"
	Commit = "abc123"
	Date = "2025-01-01"

	output, err := captureOutput(func() error {
		vcmd := &VersionCmd{}
		return vcmd.Run()
	})

	require.NoError(t, err)
	assert.Contains(t, output, "v1.2.3")
	assert.Contains(t, output, "abc123")
	assert.Contains(t, output, "2025-01-01")
}
