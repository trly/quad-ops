package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseUnit(t *testing.T) {
	// Test container unit
	containerUnit := BaseUnit{
		Name:     "test-container",
		UnitType: "container",
	}

	// Test GetServiceName for container
	assert.Equal(t, "test-container.service", containerUnit.GetServiceName())
	// Test GetUnitType
	assert.Equal(t, "container", containerUnit.GetUnitType())
	// Test GetUnitName
	assert.Equal(t, "test-container", containerUnit.GetUnitName())

	// Test volume unit
	volumeUnit := BaseUnit{
		Name:     "test-volume",
		UnitType: "volume",
	}

	// Test GetServiceName for non-container unit
	assert.Equal(t, "test-volume-volume.service", volumeUnit.GetServiceName())
	// Test GetUnitType
	assert.Equal(t, "volume", volumeUnit.GetUnitType())
	// Test GetUnitName
	assert.Equal(t, "test-volume", volumeUnit.GetUnitName())

	// Test network unit
	networkUnit := BaseUnit{
		Name:     "test-network",
		UnitType: "network",
	}

	// Test GetServiceName for non-container unit
	assert.Equal(t, "test-network-network.service", networkUnit.GetServiceName())
	// Test GetUnitType
	assert.Equal(t, "network", networkUnit.GetUnitType())
	// Test GetUnitName
	assert.Equal(t, "test-network", networkUnit.GetUnitName())
}
