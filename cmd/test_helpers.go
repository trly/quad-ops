package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// ExecuteCommandWithCapture executes a cobra command and captures all output (stdout/stderr).
// This handles both cmd.Print* and fmt.Print* outputs by redirecting os.Stdout/os.Stderr.
func ExecuteCommandWithCapture(t *testing.T, cmd *cobra.Command, args []string) (output string, err error) {
	t.Helper()

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Also set cobra's output (for cmd.Print* methods)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)

	// Channel to capture output
	outputCh := make(chan string, 1)

	// Read from pipe in goroutine
	go func() {
		var output bytes.Buffer
		_, _ = io.Copy(&output, r)
		outputCh <- output.String()
	}()

	// Execute command
	err = cmd.Execute()

	// Restore stdout/stderr
	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Get captured output
	capturedOutput := <-outputCh

	// Combine both outputs (pipe capture + cobra buffer)
	combined := capturedOutput + buf.String()

	return combined, err
}

// ExecuteCommand is a simpler helper for commands that don't need output capture.
func ExecuteCommand(t *testing.T, cmd *cobra.Command, args []string) error {
	t.Helper()
	cmd.SetArgs(args)
	return cmd.Execute()
}

// AssertCommandSuccess verifies a command executed successfully.
func AssertCommandSuccess(t *testing.T, cmd *cobra.Command, args []string) {
	t.Helper()
	err := ExecuteCommand(t, cmd, args)
	assert.NoError(t, err)
}

// AssertCommandOutput verifies command output contains expected strings.
func AssertCommandOutput(t *testing.T, cmd *cobra.Command, args []string, expectedOutputs ...string) {
	t.Helper()
	output, err := ExecuteCommandWithCapture(t, cmd, args)
	assert.NoError(t, err)

	for _, expected := range expectedOutputs {
		assert.Contains(t, output, expected, "Expected output to contain: %s\nActual output: %s", expected, output)
	}
}

// AssertCommandFailure verifies a command fails with expected error.
func AssertCommandFailure(t *testing.T, cmd *cobra.Command, args []string, expectedError string) {
	t.Helper()
	_, err := ExecuteCommandWithCapture(t, cmd, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedError)
}

// SetupCommandContext creates a command with app context for testing.
func SetupCommandContext(cmd *cobra.Command, app *App) {
	ctx := context.WithValue(context.Background(), appContextKey, app)
	cmd.SetContext(ctx)
}
