package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckConversion(t *testing.T) {
	// Prepare health check configuration for the container
	container := NewContainer("test-web")
	container.Image = "nginx:latest"
	// Initialize RunInit to avoid nil pointer dereference
	container.RunInit = new(bool)
	*container.RunInit = true
	container.HealthCmd = []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"}
	container.HealthInterval = "10s"
	container.HealthTimeout = "5s"
	container.HealthRetries = 3
	container.HealthStartPeriod = "30s"
	container.HealthStartInterval = "5s"

	// Create a Quadlet unit with the container
	quadletUnit := QuadletUnit{
		Name:      "test-web",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := GenerateQuadletUnit(quadletUnit)

	// Check that health check settings are properly included
	assert.Contains(t, unitFile, "HealthCmd=CMD-SHELL curl -f http://localhost/ || exit 1")
	assert.Contains(t, unitFile, "HealthInterval=10s")
	assert.Contains(t, unitFile, "HealthTimeout=5s")
	assert.Contains(t, unitFile, "HealthRetries=3")
	assert.Contains(t, unitFile, "HealthStartPeriod=30s")
	assert.Contains(t, unitFile, "HealthStartupInterval=5s")
}

func TestDisabledHealthCheck(t *testing.T) {
	// Create a test service with disabled health check
	service := types.ServiceConfig{
		Name:  "db",
		Image: "postgres:latest",
		HealthCheck: &types.HealthCheckConfig{
			Disable: true,
			Test:    []string{"CMD-SHELL", "pg_isready"},
		},
	}

	// Convert to container unit
	container := NewContainer("test-db")
	container.FromComposeService(service, "test")

	// Create a Quadlet unit with the container
	quadletUnit := QuadletUnit{
		Name:      "test-db",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := GenerateQuadletUnit(quadletUnit)

	// Check that health check settings are NOT included
	assert.NotContains(t, unitFile, "HealthCmd")
	assert.NotContains(t, unitFile, "HealthInterval")
}
