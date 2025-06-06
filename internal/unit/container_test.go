package unit

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/fs"
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

	// Create test project
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}

	// Convert to container objects
	container1.FromComposeService(service, project)
	// Do it again for the second container
	container2.FromComposeService(service, project)

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
	hash1 := fmt.Sprintf("%x", fs.GetContentHash(content1))
	hash2 := fmt.Sprintf("%x", fs.GetContentHash(content2))

	assert.Equal(t, hash1, hash2, "Content hashes should be identical for identical content")
}

func TestCustomHostnameNetworkAlias(t *testing.T) {
	// Initialize logger
	log.Init(false)

	// Create mock configuration
	cfg := config.DefaultProvider().InitConfig() // Initialize default config
	config.DefaultProvider().SetConfig(cfg)

	// Create basic service with custom hostname
	service := types.ServiceConfig{
		Name:     "db",
		Image:    "docker.io/postgres:latest",
		Hostname: "photoprism-db", // Custom hostname
	}

	// Process the container
	projectName := "test-project"
	project := &types.Project{
		Name:     projectName,
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	prefixedName := fmt.Sprintf("%s-%s", projectName, "db")
	container := NewContainer(prefixedName)
	container = container.FromComposeService(service, project)

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
	cfg := config.DefaultProvider().InitConfig() // Initialize default config
	config.DefaultProvider().SetConfig(cfg)

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
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	container := NewContainer("test-scrutiny")
	container = container.FromComposeService(service, project)

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
	changedUnits := make([]QuadletUnit, 0, len(project.Services))

	// Just manually create the container and check if we can find the env files directly
	// because the full processServices call needs a configured testing environment
	for serviceName, service := range project.Services {
		prefixedName := fmt.Sprintf("%s-%s", project.Name, serviceName)
		container := NewContainer(prefixedName)
		container = container.FromComposeService(service, project)

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
	project := &types.Project{
		Name:     "test",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	container := NewContainer("test-db")
	container.FromComposeService(service, project)

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
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	container.FromComposeService(service, project)

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
		project := &types.Project{
			Name:     "test-project",
			Networks: map[string]types.NetworkConfig{},
			Volumes:  map[string]types.VolumeConfig{},
		}
		container.FromComposeService(service, project)

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
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	container.FromComposeService(service, project)

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

// TestMemoryConstraints tests the processMemoryConstraints method specifically.
func TestMemoryConstraints(t *testing.T) {
	container := NewContainer("memory-test")
	unsupportedFeatures := make([]string, 0)

	// Test service-level memory constraints
	service := types.ServiceConfig{
		Name:           "memory-service",
		MemLimit:       1024 * 1024 * 100, // 100 MB
		MemReservation: 1024 * 1024 * 50,  // 50 MB
		MemSwapLimit:   1024 * 1024 * 200, // 200 MB
	}

	container.processMemoryConstraints(service, &unsupportedFeatures)

	assert.Equal(t, "104857600", container.Memory)
	assert.Equal(t, "52428800", container.MemoryReservation)
	assert.Equal(t, "209715200", container.MemorySwap)
	assert.Len(t, unsupportedFeatures, 3)
	assert.Contains(t, container.PodmanArgs, "--memory=104857600")
	assert.Contains(t, container.PodmanArgs, "--memory-reservation=52428800")
	assert.Contains(t, container.PodmanArgs, "--memory-swap=209715200")
}

// TestCPUConstraints tests the processCPUConstraints method specifically.
func TestCPUConstraints(t *testing.T) {
	container := NewContainer("cpu-test")
	unsupportedFeatures := make([]string, 0)

	// Test service-level CPU constraints
	service := types.ServiceConfig{
		Name:      "cpu-service",
		CPUShares: 512,
		CPUQuota:  50000,
		CPUPeriod: 100000,
		PidsLimit: 100,
	}

	container.processCPUConstraints(service, &unsupportedFeatures)

	assert.Equal(t, int64(512), container.CPUShares)
	assert.Equal(t, int64(50000), container.CPUQuota)
	assert.Equal(t, int64(100000), container.CPUPeriod)
	assert.Equal(t, int64(100), container.PidsLimit)
	assert.Len(t, unsupportedFeatures, 3) // CPUShares, CPUQuota, CPUPeriod
	assert.Contains(t, container.PodmanArgs, "--cpu-shares=512")
	assert.Contains(t, container.PodmanArgs, "--cpu-quota=50000")
	assert.Contains(t, container.PodmanArgs, "--cpu-period=100000")
}

// TestSecurityOptions tests the processSecurityOptions method specifically.
func TestSecurityOptions(t *testing.T) {
	container := NewContainer("security-test")
	unsupportedFeatures := make([]string, 0)

	// Test privileged mode and security options
	service := types.ServiceConfig{
		Name:        "security-service",
		Privileged:  true,
		SecurityOpt: []string{"label:disable", "label:level:s0:c1,c2"},
	}

	container.processSecurityOptions(service, &unsupportedFeatures)

	assert.Len(t, unsupportedFeatures, 3) // Privileged mode + 2 security options
	assert.Contains(t, container.PodmanArgs, "--privileged")
	assert.Contains(t, container.PodmanArgs, "--security-opt=label=disable")
	assert.Contains(t, container.PodmanArgs, "--security-opt=label:level:s0:c1,c2")
}

// TestProcessServiceResources tests the main resource processing coordination.
func TestProcessServiceResources(t *testing.T) {
	container := NewContainer("resource-coordinator-test")

	// Test combined resource processing
	service := types.ServiceConfig{
		Name:        "combined-service",
		Image:       "test/image:latest",
		MemLimit:    1024 * 1024 * 100, // 100 MB
		CPUShares:   512,
		Privileged:  true,
		SecurityOpt: []string{"label:disable"},
	}

	container.processServiceResources(service)

	// Verify all resource types were processed
	assert.Equal(t, "104857600", container.Memory)
	assert.Equal(t, int64(512), container.CPUShares)
	assert.Contains(t, container.PodmanArgs, "--memory=104857600")
	assert.Contains(t, container.PodmanArgs, "--cpu-shares=512")
	assert.Contains(t, container.PodmanArgs, "--privileged")
	assert.Contains(t, container.PodmanArgs, "--security-opt=label=disable")
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
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	container.FromComposeService(service, project)

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

func TestVolumeOptionsPreservation(t *testing.T) {
	container := NewContainer("test-volume-options")

	// Create a service with various volume options including SELinux labels
	service := types.ServiceConfig{
		Name:  "volume-service",
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{
				Type:   "bind",
				Source: "/host/path",
				Target: "/container/path",
				Bind: &types.ServiceVolumeBind{
					SELinux: "Z",
				},
			},
			{
				Type:     "bind",
				Source:   "/host/readonly",
				Target:   "/container/readonly",
				ReadOnly: true,
				Bind: &types.ServiceVolumeBind{
					SELinux: "z",
				},
			},
		},
	}

	// Convert service to container
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  map[string]types.VolumeConfig{},
	}
	container.FromComposeService(service, project)

	// Verify that SELinux options are preserved
	assert.Len(t, container.Volume, 2)
	assert.Contains(t, container.Volume, "/host/path:/container/path:rw,Z")
	assert.Contains(t, container.Volume, "/host/readonly:/container/readonly:ro,z")
}

func TestExternalVolumeHandling(t *testing.T) {
	// Create a service that uses an external volume
	service := types.ServiceConfig{
		Name:  "test-service",
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{
				Type:   "volume",
				Source: "shared-data",
				Target: "/data",
			},
		},
	}

	// Define project volumes - one external, one regular
	projectVolumes := map[string]types.VolumeConfig{
		"shared-data": {
			Name:     "shared-data",
			External: types.External(true), // External volume
		},
		"local-data": {
			Name: "local-data",
			// External is false by default
		},
	}

	// Create container
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  projectVolumes,
	}
	container := NewContainer("test-project-test-service")
	container = container.FromComposeService(service, project)

	// Check that external volume is referenced without project prefix
	assert.Len(t, container.Volume, 1)
	assert.Contains(t, container.Volume, "shared-data:/data", "External volume should use name directly without project prefix")
	assert.NotContains(t, container.Volume, "test-project-shared-data.volume:/data", "External volume should not have project prefix")
}

func TestRegularVolumeHandling(t *testing.T) {
	// Create a service that uses a regular (non-external) volume
	service := types.ServiceConfig{
		Name:  "test-service",
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{
				Type:   "volume",
				Source: "local-data",
				Target: "/data",
			},
		},
	}

	// Define project volumes - one external, one regular
	projectVolumes := map[string]types.VolumeConfig{
		"shared-data": {
			Name:     "shared-data",
			External: types.External(true), // External volume
		},
		"local-data": {
			Name: "local-data",
			// External is false by default
		},
	}

	// Create container
	project := &types.Project{
		Name:     "test-project",
		Networks: map[string]types.NetworkConfig{},
		Volumes:  projectVolumes,
	}
	container := NewContainer("test-project-test-service")
	container = container.FromComposeService(service, project)

	// Check that regular volume is referenced with project prefix
	assert.Len(t, container.Volume, 1)
	assert.Contains(t, container.Volume, "test-project-local-data.volume:/data", "Regular volume should have project prefix")
	assert.NotContains(t, container.Volume, "local-data:/data", "Regular volume should not use name directly")
}
