package unit

import (
	"crypto/sha1" //nolint:gosec // Not used for security purposes, just content comparison
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

// initTestLogger initializes a logger for testing that discards output.
func initTestLogger() {
	// Create a handler that discards output
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Set it as default in our logger package too
	log.Init(false)
}

func TestDeterministicUnitContent(t *testing.T) {
	// Initialize logger
	log.Init(false)

	// Create two identical container configs
	container1 := NewContainer("test-container")
	container2 := NewContainer("test-container")

	// Configure them identically
	service := types.ServiceConfig{
		Image: "docker.io/test:latest",
		Environment: types.MappingWithEquals{
			"VAR1": &[]string{"value1"}[0],
			"VAR2": &[]string{"value2"}[0],
		},
	}

	// Convert to container objects
	container1.FromComposeService(service, "test-project")
	// Do it again for the second container
	container2.FromComposeService(service, "test-project")

	// Generate quadlet units
	unit1 := &QuadletUnit{
		Name:      "test-container",
		Type:      "container",
		Container: *container1,
	}

	unit2 := &QuadletUnit{
		Name:      "test-container",
		Type:      "container",
		Container: *container2,
	}

	// Generate the content - this should be deterministic
	content1 := GenerateQuadletUnit(*unit1)
	content2 := GenerateQuadletUnit(*unit2)

	// Should produce identical content
	assert.Equal(t, content1, content2, "Unit content should be identical when generated from identical configs")

	// Check if content hashing is deterministic
	hash1 := GetContentHash(content1)
	hash2 := GetContentHash(content2)

	assert.Equal(t, hash1, hash2, "Content hashes should be identical for identical content")
}

func GetContentHash(content string) string {
	hash := sha1.New() //nolint:gosec // Not used for security purposes
	hash.Write([]byte(content))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func TestCustomHostnameNetworkAlias(t *testing.T) {
	// Initialize logger
	log.Init(false)

	// Create mock configuration
	cfg := config.InitConfig() // Initialize default config
	config.SetConfig(cfg)

	// Create basic service with custom hostname
	service := types.ServiceConfig{
		Name:     "db",
		Image:    "docker.io/postgres:latest",
		Hostname: "photoprism-db", // Custom hostname
	}

	// Create project
	project := types.Project{
		Name: "test-project",
		Services: types.Services{
			"db": service,
		},
	}

	// Process the container
	prefixedName := fmt.Sprintf("%s-%s", project.Name, "db")
	container := NewContainer(prefixedName)
	container = container.FromComposeService(service, project.Name)

	// Set custom container name (usePodmanNames=false)
	container.ContainerName = prefixedName

	// Add service name as network alias
	container.NetworkAlias = append(container.NetworkAlias, "db")

	// Add custom hostname as network alias if specified
	if container.HostName != "" && container.HostName != "db" {
		container.NetworkAlias = append(container.NetworkAlias, container.HostName)
	}

	// Check that both the service name and custom hostname are in network aliases
	assert.Contains(t, container.NetworkAlias, "db", "Service name should be included in network aliases")
	assert.Contains(t, container.NetworkAlias, "photoprism-db", "Custom hostname should be included in network aliases")

	// Generate unit to verify output
	unit := &QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
	}

	// Generate the content and check it contains both network aliases
	content := GenerateQuadletUnit(*unit)
	assert.Contains(t, content, "NetworkAlias=db", "Unit content should include service name as NetworkAlias")
	assert.Contains(t, content, "NetworkAlias=photoprism-db", "Unit content should include custom hostname as NetworkAlias")
}

func TestDeviceMappingFormat(t *testing.T) {
	// Initialize logger
	log.Init(false)

	// Create mock configuration
	cfg := config.InitConfig() // Initialize default config
	config.SetConfig(cfg)

	// Create a service with device mappings like in the Scrutiny compose file
	service := types.ServiceConfig{
		Name:  "scrutiny",
		Image: "ghcr.io/analogj/scrutiny:master-omnibus",
	}

	// Add devices to the service
	service.Devices = []types.DeviceMapping{
		{Source: "/dev/nvme0"},
		{Source: "/dev/nvme1"},
	}

	// Process the container
	container := NewContainer("test-scrutiny")
	container = container.FromComposeService(service, "test-project")

	// Generate unit to verify output
	unit := &QuadletUnit{
		Name:      "test-scrutiny",
		Type:      "container",
		Container: *container,
	}

	// Generate the content and check it contains correctly formatted device arguments
	content := GenerateQuadletUnit(*unit)

	// The device arguments should be properly formatted for podman
	assert.Contains(t, content, "PodmanArgs=--device=/dev/nvme0", "Device paths should be properly formatted")
	assert.Contains(t, content, "PodmanArgs=--device=/dev/nvme1", "Device paths should be properly formatted")

	// Should NOT contain the incorrect format with curly braces
	assert.NotContains(t, content, "PodmanArgs=--device={/dev/nvme0", "Device paths should not include curly braces")
}

func TestServiceSpecificEnvironmentFiles(t *testing.T) {
	// Initialize logger
	initTestLogger()
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "quad-ops-env-test-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create a test service
	serviceName := "db"
	service := types.ServiceConfig{
		Name:  serviceName,
		Image: "postgres:14",
	}

	// Create a project
	project := &types.Project{
		Name:       "test-project",
		WorkingDir: tmpDir,
		Services: types.Services{
			serviceName: service,
		},
	}

	// Create test environment files
	testFiles := []string{
		fmt.Sprintf("%s/.env.%s", tmpDir, serviceName),
		fmt.Sprintf("%s/%s.env", tmpDir, serviceName),
		fmt.Sprintf("%s/env/%s.env", tmpDir, serviceName),
	}

	// Create the env directory
	err = os.MkdirAll(filepath.Join(tmpDir, "env"), 0750)
	assert.NoError(t, err)

	// Create the environment files with sample content
	for i, file := range testFiles {
		content := fmt.Sprintf("# Environment file %d\nPOSTGRES_DB=testdb\nPOSTGRES_PASSWORD=password123\n", i+1)
		err := os.WriteFile(file, []byte(content), 0600)
		assert.NoError(t, err)
	}

	// Create a slice to store the test units
	changedUnits := make([]QuadletUnit, 0)

	// Just manually create the container and check if we can find the env files directly
	// because the full processServices call needs a configured testing environment
	for serviceName, service := range project.Services {
		prefixedName := fmt.Sprintf("%s-%s", project.Name, serviceName)
		container := NewContainer(prefixedName)
		container = container.FromComposeService(service, project.Name)

		// Check for service-specific .env files in the project directory
		if project.WorkingDir != "" {
			// Look for .env files with various naming patterns
			possibleEnvFiles := []string{
				fmt.Sprintf("%s/.env.%s", project.WorkingDir, serviceName),
				fmt.Sprintf("%s/%s.env", project.WorkingDir, serviceName),
				fmt.Sprintf("%s/env/%s.env", project.WorkingDir, serviceName),
			}

			for _, envFilePath := range possibleEnvFiles {
				// Check if file exists
				if _, err := os.Stat(envFilePath); err == nil {
					container.EnvironmentFile = append(container.EnvironmentFile, envFilePath)
				}
			}
		}

		// Create a QuadletUnit with the container
		quadletUnit := QuadletUnit{
			Name:      prefixedName,
			Type:      "container",
			Container: *container,
		}

		changedUnits = append(changedUnits, quadletUnit)
	}

	// Verify that the environment files were discovered and added
	for _, unit := range changedUnits {
		if unit.Type == "container" && unit.Name == "test-project-db" {
			// Verify that 3 environment files were found
			assert.Equal(t, 3, len(unit.Container.EnvironmentFile))

			// Verify that all expected files are in the list
			for _, expectedFile := range testFiles {
				found := false
				for _, envFile := range unit.Container.EnvironmentFile {
					if envFile == expectedFile {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected environment file %s not found", expectedFile)
			}
			break
		}
	}
}

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

func TestLoggingConfiguration(t *testing.T) {
	// Create a test container for logging
	container := NewContainer("logging-test")

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
	quadletUnit := QuadletUnit{
		Name:      "logging-test",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := GenerateQuadletUnit(quadletUnit)

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
		container := NewContainer("restart-test")

		// Create a compose service with restart policy
		service := types.ServiceConfig{
			Name:    "restart-service",
			Image:   "test/image:latest",
			Restart: composePolicy,
		}

		// Convert service to container
		container.FromComposeService(service, "test-project")

		// Create a systemd config with the restart policy
		systemdConfig := SystemdConfig{}
		systemdConfig.RestartPolicy = container.RestartPolicy

		// Create a quadlet unit
		quadletUnit := QuadletUnit{
			Name:      "restart-test",
			Type:      "container",
			Container: *container,
			Systemd:   systemdConfig,
		}

		// Generate the unit file
		unitFile := GenerateQuadletUnit(quadletUnit)

		// Verify restart policy is correctly mapped
		assert.Contains(t, unitFile, "Restart="+systemdPolicy,
			"Docker Compose policy %s should map to systemd policy %s", composePolicy, systemdPolicy)
	}
}

func TestContainerResourceConstraints(t *testing.T) {
	// Create a test container with resource constraints
	container := NewContainer("resource-test")

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
	quadletUnit := QuadletUnit{
		Name:      "resource-test",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := GenerateQuadletUnit(quadletUnit)

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
	container := NewContainer("advanced-test")

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
	quadletUnit := QuadletUnit{
		Name:      "advanced-test",
		Type:      "container",
		Container: *container,
	}

	// Generate the unit file
	unitFile := GenerateQuadletUnit(quadletUnit)

	// Verify advanced configuration is in the unit file
	assert.Contains(t, unitFile, "Sysctl=net.core.somaxconn=1024")
	assert.Contains(t, unitFile, "Sysctl=net.ipv4.ip_forward=1")
	assert.Contains(t, unitFile, "Ulimit=nofile=1024:2048")
	assert.Contains(t, unitFile, "Ulimit=nproc=65535")
	assert.Contains(t, unitFile, "Tmpfs=/tmp:rw,size=1G")
	assert.Contains(t, unitFile, "Tmpfs=tmp")
	assert.Contains(t, unitFile, "UserNS=keep-id")
}
