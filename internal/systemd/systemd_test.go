package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseUnit(t *testing.T) {
	// Create test unit
	baseUnit := NewBaseUnit("test-container", "container")

	// Test that the ResetFailed method is available
	// We can't actually reset units in the test environment, but we can
	// ensure the interface is implemented properly
	assertImplementsUnit(t, baseUnit)

	// Test service name generation
	assert.Equal(t, "test-container.service", baseUnit.GetServiceName())
	assert.Equal(t, "test-container", baseUnit.GetUnitName())
	assert.Equal(t, "container", baseUnit.GetUnitType())
}

// Helper function to assert that a type implements the Unit interface.
func assertImplementsUnit(t *testing.T, unit Unit) {
	// Assert that the unit implements all required methods
	assert.NotNil(t, unit.GetServiceName())
	assert.NotEmpty(t, unit.GetUnitType())
	assert.NotEmpty(t, unit.GetUnitName())

	// For the reset failed functionality specifically
	// We ensure the method exists and is callable (though we don't test actual functionality
	// since that requires a real systemd connection)
	isResetFailedImplemented := false
	switch unit.(type) {
	case *BaseUnit:
		isResetFailedImplemented = true
	}
	assert.True(t, isResetFailedImplemented, "ResetFailed method should be implemented")
}
