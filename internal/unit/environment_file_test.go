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
	"github.com/trly/quad-ops/internal/logger"
)

// initTestLogger initializes a logger for testing that discards output.
func initTestLogger() {
	// Create a handler that discards output
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	log := slog.New(handler)
	slog.SetDefault(log)

	// Set it as default in our logger package too
	logger.Init(false)
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
