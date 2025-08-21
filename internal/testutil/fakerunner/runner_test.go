package fakerunner

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFakeRunner(t *testing.T) {
	t.Run("new runner starts empty", func(t *testing.T) {
		runner := New()
		assert.Empty(t, runner.GetCalls())
	})

	t.Run("set and get output", func(t *testing.T) {
		runner := New()
		expectedOutput := []byte("test output")

		runner.SetOutput("echo", []string{"hello"}, expectedOutput)
		output, err := runner.CombinedOutput(context.Background(), "echo", "hello")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("set and get error", func(t *testing.T) {
		runner := New()
		expectedErr := errors.New("test error")

		runner.SetError("failing-command", []string{}, expectedErr)
		output, err := runner.CombinedOutput(context.Background(), "failing-command")

		assert.Nil(t, output)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("captures calls", func(t *testing.T) {
		runner := New()

		_, _ = runner.CombinedOutput(context.Background(), "echo", "hello")
		_, _ = runner.CombinedOutput(context.Background(), "ls", "-la")

		calls := runner.GetCalls()
		assert.Len(t, calls, 2)
		assert.Equal(t, "echo", calls[0].Name)
		assert.Equal(t, []string{"hello"}, calls[0].Args)
		assert.Equal(t, "ls", calls[1].Name)
		assert.Equal(t, []string{"-la"}, calls[1].Args)
	})

	t.Run("default behavior returns empty output", func(t *testing.T) {
		runner := New()

		output, err := runner.CombinedOutput(context.Background(), "unknown-command")

		assert.NoError(t, err)
		assert.Empty(t, output)
	})

	t.Run("reset clears state", func(t *testing.T) {
		runner := New()

		runner.SetOutput("echo", []string{"test"}, []byte("output"))
		runner.SetError("fail", []string{}, errors.New("error"))
		_, _ = runner.CombinedOutput(context.Background(), "echo", "test")

		runner.Reset()

		assert.Empty(t, runner.GetCalls())

		// After reset, should return default behavior
		output, err := runner.CombinedOutput(context.Background(), "echo", "test")
		assert.NoError(t, err)
		assert.Empty(t, output)
	})
}
