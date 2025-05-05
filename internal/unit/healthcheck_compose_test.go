package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectHealthCheckImplementation(t *testing.T) {
	// Directly test our conversion implementation by manually setting health check fields
	container := &Container{
		BaseUnit: BaseUnit{
			Name:     "test-web",
			UnitType: "container",
		},
		Image:               "nginx:latest",
		HealthCmd:           []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
		HealthInterval:      "10s",
		HealthTimeout:       "5s",
		HealthRetries:       3,
		HealthStartPeriod:   "30s",
		HealthStartInterval: "5s",
		RunInit:             new(bool),
	}
	*container.RunInit = true

	// Create a Quadlet unit with the container
	quadletUnit := QuadletUnit{
		Name:      "test-web",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := GenerateQuadletUnit(quadletUnit)

	// Check that health check settings are properly included in the unit file
	assert.Contains(t, unitFile, "HealthCmd=CMD-SHELL curl -f http://localhost/ || exit 1")
	assert.Contains(t, unitFile, "HealthInterval=10s")
	assert.Contains(t, unitFile, "HealthTimeout=5s")
	assert.Contains(t, unitFile, "HealthRetries=3")
	assert.Contains(t, unitFile, "HealthStartPeriod=30s")
	assert.Contains(t, unitFile, "HealthStartupInterval=5s")
}
