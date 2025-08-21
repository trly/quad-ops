package execx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealRunner_CombinedOutput(t *testing.T) {
	runner := NewRealRunner()
	ctx := context.Background()

	t.Run("successful command execution", func(t *testing.T) {
		output, err := runner.CombinedOutput(ctx, "echo", "hello", "world")
		require.NoError(t, err)
		assert.Contains(t, string(output), "hello world")
	})

	t.Run("command not found", func(t *testing.T) {
		_, err := runner.CombinedOutput(ctx, "nonexistent-command-12345")
		assert.Error(t, err)
	})

	t.Run("command with error exit code", func(t *testing.T) {
		_, err := runner.CombinedOutput(ctx, "sh", "-c", "exit 1")
		assert.Error(t, err)
	})
}
