package unit_test

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/unit"
)

func TestContainerResourceConstraints(t *testing.T) {
	// Create a test container with resource constraints
	container := unit.NewContainer("resource-test")

	// Create a compose service with resource constraints
	service := types.ServiceConfig{
		Name:      "resource-service",
		Image:     "test/image:latest",
		MemLimit:  1024 * 1024 * 100, // 100 MB
		CPUShares: 512,
		CPUQuota:  50000,
		CPUPeriod: 100000,
		PidsLimit: 100,
	}

	// Convert service to container
	container.FromComposeService(service, "test-project")

	// Create a quadlet unit
	quadletUnit := unit.QuadletUnit{
		Name:      "resource-test",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := unit.GenerateQuadletUnit(quadletUnit)

	// Verify resource constraints are in the unit file
	// Memory is not supported by Podman Quadlet, so we don't include it in the unit file
	// assert.Contains(t, unitFile, "Memory=104857600")
	// CPU directives are not supported by Podman Quadlet, so we don't include them in the unit file
	// assert.Contains(t, unitFile, "CPUShares=512")
	// assert.Contains(t, unitFile, "CPUQuota=50000")
	// assert.Contains(t, unitFile, "CPUPeriod=100000")
	assert.Contains(t, unitFile, "PidsLimit=100")
}

func TestContainerAdvancedConfig(t *testing.T) {
	// Create a test container with advanced configuration
	container := unit.NewContainer("advanced-test")

	// Create sysctls mapping
	sysctls := types.Mapping{
		"net.ipv4.ip_forward": "1",
		"net.core.somaxconn":  "1024",
	}

	ulimits := map[string]*types.UlimitsConfig{
		"nofile": {
			Soft: 1024,
			Hard: 2048,
		},
		"nproc": {
			Soft: 65535,
			Hard: 65535,
		},
	}

	// Create a compose service with advanced configuration
	service := types.ServiceConfig{
		Name:       "advanced-service",
		Image:      "test/image:latest",
		Sysctls:    sysctls,
		Ulimits:    ulimits,
		Tmpfs:      types.StringList{"tmp", "/tmp:rw,size=1G"},
		UserNSMode: "keep-id",
	}

	// Convert service to container
	container.FromComposeService(service, "test-project")

	// Create a quadlet unit
	quadletUnit := unit.QuadletUnit{
		Name:      "advanced-test",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := unit.GenerateQuadletUnit(quadletUnit)

	// Verify advanced configuration is in the unit file
	assert.Contains(t, unitFile, "Sysctl=net.core.somaxconn=1024")
	assert.Contains(t, unitFile, "Sysctl=net.ipv4.ip_forward=1")
	assert.Contains(t, unitFile, "Ulimit=nofile=1024:2048")
	assert.Contains(t, unitFile, "Ulimit=nproc=65535")
	assert.Contains(t, unitFile, "Tmpfs=/tmp:rw,size=1G")
	assert.Contains(t, unitFile, "Tmpfs=tmp")
	assert.Contains(t, unitFile, "UserNS=keep-id")
}
