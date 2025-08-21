package systemd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		originalErr := errors.New("connection refused")
		err := NewError("Start", "test-unit", "container", originalErr)

		expected := "systemd Start failed for test-unit.container: connection refused"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		originalErr := errors.New("connection refused")
		err := NewError("Start", "test-unit", "container", originalErr)

		assert.Equal(t, originalErr, errors.Unwrap(err))
	})

	t.Run("IsError detects Error", func(t *testing.T) {
		originalErr := errors.New("connection refused")
		err := NewError("Start", "test-unit", "container", originalErr)

		assert.True(t, IsError(err))
		assert.False(t, IsError(originalErr))
	})
}

func TestConnectionError(t *testing.T) {
	t.Run("Error returns formatted message for user mode", func(t *testing.T) {
		originalErr := errors.New("permission denied")
		err := NewConnectionError(true, originalErr)

		expected := "failed to connect to systemd user bus: permission denied"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error returns formatted message for system mode", func(t *testing.T) {
		originalErr := errors.New("permission denied")
		err := NewConnectionError(false, originalErr)

		expected := "failed to connect to systemd system bus: permission denied"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		originalErr := errors.New("permission denied")
		err := NewConnectionError(true, originalErr)

		assert.Equal(t, originalErr, errors.Unwrap(err))
	})

	t.Run("IsConnectionError detects ConnectionError", func(t *testing.T) {
		originalErr := errors.New("permission denied")
		err := NewConnectionError(true, originalErr)

		assert.True(t, IsConnectionError(err))
		assert.False(t, IsConnectionError(originalErr))
	})
}

func TestUnitNotFoundError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := NewUnitNotFoundError("test-unit", "container")

		expected := "unit not found: test-unit.container"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("IsUnitNotFoundError detects UnitNotFoundError", func(t *testing.T) {
		err := NewUnitNotFoundError("test-unit", "container")
		otherErr := errors.New("some other error")

		assert.True(t, IsUnitNotFoundError(err))
		assert.False(t, IsUnitNotFoundError(otherErr))
	})
}

func TestErrorTypeCheckers(t *testing.T) {
	t.Run("IsError returns false for non-Error", func(t *testing.T) {
		err := errors.New("some random error")
		assert.False(t, IsError(err))

		connErr := NewConnectionError(true, err)
		assert.False(t, IsError(connErr))
	})

	t.Run("IsConnectionError returns false for non-ConnectionError", func(t *testing.T) {
		err := errors.New("some random error")
		assert.False(t, IsConnectionError(err))

		systemdErr := NewError("Start", "unit", "type", err)
		assert.False(t, IsConnectionError(systemdErr))
	})

	t.Run("IsUnitNotFoundError returns false for non-UnitNotFoundError", func(t *testing.T) {
		err := errors.New("some random error")
		assert.False(t, IsUnitNotFoundError(err))

		systemdErr := NewError("Start", "unit", "type", err)
		assert.False(t, IsUnitNotFoundError(systemdErr))
	})
}
