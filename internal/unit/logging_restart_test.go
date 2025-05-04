package unit_test

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/unit"
)

func TestLoggingConfiguration(t *testing.T) {
	// Create a test container for logging
	container := unit.NewContainer("logging-test")

	// Create a compose service with logging config
	logOpts := map[string]string{
		"max-size": "10m",
		"max-file": "3",
	}

	service := types.ServiceConfig{
		Name:      "logging-service",
		Image:     "test/image:latest",
		LogDriver: "json-file",
		LogOpt:    logOpts,
	}

	// Convert service to container
	container.FromComposeService(service, "test-project")

	// Create a quadlet unit
	quadletUnit := unit.QuadletUnit{
		Name:      "logging-test",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := unit.GenerateQuadletUnit(quadletUnit)

	// Verify logging configuration is in the unit file
	assert.Contains(t, unitFile, "LogDriver=json-file")
	assert.Contains(t, unitFile, "LogOpt=max-file=3")
	assert.Contains(t, unitFile, "LogOpt=max-size=10m")
}

func TestRestartPolicy(t *testing.T) {
	// Test different restart policies
	policies := map[string]string{
		"no":             "no",
		"always":         "always",
		"on-failure":     "on-failure",
		"unless-stopped": "always", // Maps to always in systemd
	}

	for composePolicy, systemdPolicy := range policies {
		// Create a test container
		container := unit.NewContainer("restart-test")

		// Create a compose service with restart policy
		service := types.ServiceConfig{
			Name:    "restart-service",
			Image:   "test/image:latest",
			Restart: composePolicy,
		}

		// Convert service to container
		container.FromComposeService(service, "test-project")

		// Create a systemd config with the restart policy
		systemdConfig := unit.SystemdConfig{}
		systemdConfig.RestartPolicy = container.RestartPolicy

		// Create a quadlet unit
		quadletUnit := unit.QuadletUnit{
			Name:      "restart-test",
			Type:      "container",
			Container: *container,
			Systemd:   systemdConfig,
		}

		// Generate the unit file
		unitFile := unit.GenerateQuadletUnit(quadletUnit)

		// Verify restart policy is correctly mapped
		assert.Contains(t, unitFile, "Restart="+systemdPolicy,
			"Docker Compose policy %s should map to systemd policy %s", composePolicy, systemdPolicy)
	}
}
