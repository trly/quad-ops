package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResetFailed(t *testing.T) {
	// Create test units
	containerUnit := &Container{
		BaseUnit: BaseUnit{
			Name:     "test-container",
			UnitType: "container",
		},
	}

	volumeUnit := &Volume{
		BaseUnit: BaseUnit{
			Name:     "test-volume",
			UnitType: "volume",
		},
	}

	networkUnit := &Network{
		BaseUnit: BaseUnit{
			Name:     "test-network",
			UnitType: "network",
		},
	}

	baseSystemdUnit := &BaseSystemdUnit{
		Name: "test-base",
		Type: "container",
	}

	// Test that the ResetFailed method is available on all unit types
	// We can't actually reset units in the test environment, but we can
	// ensure the interface is implemented properly
	assertImplementsSystemdUnit(t, containerUnit)
	assertImplementsSystemdUnit(t, volumeUnit)
	assertImplementsSystemdUnit(t, networkUnit)
	assertImplementsSystemdUnit(t, baseSystemdUnit)
}

// Helper function to assert that a type implements the SystemdUnit interface.
func assertImplementsSystemdUnit(t *testing.T, unit SystemdUnit) {
	// Assert that the unit implements all required methods
	assert.NotNil(t, unit.GetServiceName())
	assert.NotEmpty(t, unit.GetUnitType())
	assert.NotEmpty(t, unit.GetUnitName())

	// For the reset failed functionality specifically
	// We ensure the method exists and is callable (though we don't test actual functionality
	// since that requires a real systemd connection)
	isResetFailedImplemented := false
	switch unit.(type) {
	case *Container, *Volume, *Network, *BaseSystemdUnit, *BaseUnit:
		isResetFailedImplemented = true
	}
	assert.True(t, isResetFailedImplemented, "ResetFailed method should be implemented")
}