package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/unit"
)

// Test addBuildDependency function specifically.
func TestAddBuildDependencyIsolated(t *testing.T) {
	logger := initTestLogger()
	processor := NewProcessor(nil, nil, nil, logger, false)

	tests := []struct {
		name             string
		serviceName      string
		preAddServices   []string
		expectError      bool
		expectDependency bool
	}{
		{
			name:             "add dependency successfully",
			serviceName:      "web",
			preAddServices:   []string{"web"},
			expectError:      false,
			expectDependency: true,
		},
		{
			name:             "build service already exists",
			serviceName:      "api",
			preAddServices:   []string{"api", "api-build"},
			expectError:      false,
			expectDependency: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depGraph := dependency.NewServiceDependencyGraph()

			// Pre-add services
			for _, serviceName := range tt.preAddServices {
				err := depGraph.AddService(serviceName)
				require.NoError(t, err)
			}

			err := processor.addBuildDependency(depGraph, tt.serviceName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectDependency {
					buildName := fmt.Sprintf("%s-build", tt.serviceName)
					deps, _ := depGraph.GetDependencies(tt.serviceName)
					assert.Contains(t, deps, buildName)
				}
			}
		})
	}
}

// Test createQuadletUnit and createInitQuadletUnit functions.
func TestCreateQuadletUnits(t *testing.T) {
	t.Run("createQuadletUnit", func(t *testing.T) {
		container := unit.NewContainer("test-container")
		container.RestartPolicy = "unless-stopped"

		quadletUnit := createQuadletUnit("test-webapp", container)

		assert.Equal(t, "test-webapp", quadletUnit.Name)
		assert.Equal(t, "container", quadletUnit.Type)
		assert.Equal(t, "unless-stopped", quadletUnit.Systemd.RestartPolicy)
	})

	t.Run("createInitQuadletUnit", func(t *testing.T) {
		container := unit.NewContainer("test-container")

		quadletUnit := createInitQuadletUnit("test-webapp-init-0", container)

		assert.Equal(t, "test-webapp-init-0", quadletUnit.Name)
		assert.Equal(t, "container", quadletUnit.Type)
		assert.Equal(t, "oneshot", quadletUnit.Systemd.Type)
		assert.True(t, quadletUnit.Systemd.RemainAfterExit)
	})
}

// Test createContainerFromService function more thoroughly.
func TestCreateContainerFromServiceDetailed(t *testing.T) {
	// Create temporary directory with various env files
	tmpDir, err := os.MkdirTemp("", "quad-ops-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create various env files to test discovery
	envFiles := map[string]string{
		".env":           "GENERAL=true",
		".env.webapp":    "SERVICE_SPECIFIC=webapp",
		"webapp.env":     "ALT_FORMAT=webapp",
		"env/webapp.env": "NESTED=webapp",
	}

	// Create env subdirectory
	err = os.MkdirAll(filepath.Join(tmpDir, "env"), 0o750)
	require.NoError(t, err)

	for filename, content := range envFiles {
		envPath := filepath.Join(tmpDir, filename)
		err = os.WriteFile(envPath, []byte(content), 0o600)
		require.NoError(t, err)
	}

	logger := initTestLogger()
	processor := NewProcessor(nil, nil, nil, logger, false)

	service := types.ServiceConfig{
		Name:     "webapp",
		Image:    "nginx:latest",
		Hostname: "custom-hostname",
		Environment: map[string]*string{
			"ENV_VAR": stringPtr("test"),
		},
		Ports: []types.ServicePortConfig{
			{
				Target:    80,
				Published: "8080",
				Protocol:  "tcp",
			},
		},
	}

	project := &types.Project{
		Name:       "test",
		WorkingDir: tmpDir,
	}

	container := processor.createContainerFromService(service, "test-webapp", "webapp", project)

	// Verify basic container properties
	assert.Equal(t, "test-webapp", container.Name)
	assert.Equal(t, "nginx:latest", container.Image)

	// Verify environment files were discovered
	assert.Greater(t, len(container.EnvironmentFile), 0)
	assert.Contains(t, container.EnvironmentFile, filepath.Join(tmpDir, ".env"))
	assert.Contains(t, container.EnvironmentFile, filepath.Join(tmpDir, ".env.webapp"))

	// Verify network aliases
	assert.Contains(t, container.NetworkAlias, "webapp")
	assert.Contains(t, container.NetworkAlias, "custom-hostname")

	// Verify container naming
	assert.Equal(t, "test-webapp", container.ContainerName)
}

// Test configureContainerNaming function scenarios.
func TestConfigureContainerNamingScenarios(t *testing.T) {
	logger := initTestLogger()
	processor := NewProcessor(nil, nil, nil, logger, false)

	tests := []struct {
		name               string
		prefixedName       string
		serviceName        string
		hostname           string
		expectedAliasCount int
		expectedAliases    []string
	}{
		{
			name:               "basic service name only",
			prefixedName:       "proj-web",
			serviceName:        "web",
			hostname:           "",
			expectedAliasCount: 1,
			expectedAliases:    []string{"web"},
		},
		{
			name:               "with different hostname",
			prefixedName:       "proj-api",
			serviceName:        "api",
			hostname:           "backend-api",
			expectedAliasCount: 2,
			expectedAliases:    []string{"api", "backend-api"},
		},
		{
			name:               "hostname same as service name",
			prefixedName:       "proj-db",
			serviceName:        "db",
			hostname:           "db",
			expectedAliasCount: 1,
			expectedAliases:    []string{"db"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := unit.NewContainer("test")
			container.HostName = tt.hostname

			processor.configureContainerNaming(container, tt.prefixedName, tt.serviceName)

			assert.Equal(t, tt.prefixedName, container.ContainerName)
			assert.Len(t, container.NetworkAlias, tt.expectedAliasCount)

			for _, expectedAlias := range tt.expectedAliases {
				assert.Contains(t, container.NetworkAlias, expectedAlias)
			}
		})
	}
}

// Helper function for string pointer.
func stringPtr(s string) *string {
	return &s
}
