package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseUnit(t *testing.T) {
	// Create test unit
	baseUnit := NewBaseUnit("test-container", "container")

	// Test service name generation
	assert.Equal(t, "test-container.service", baseUnit.GetServiceName())
	assert.Equal(t, "test-container", baseUnit.GetUnitName())
	assert.Equal(t, "container", baseUnit.GetUnitType())
}
