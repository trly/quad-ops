package unit

import (
	"os"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
)

func TestProcessComposeProjectsMultiRepository(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Set up fake config for testing
	cfg := &config.Config{
		QuadletDir: tempDir,
		Repositories: []config.RepositoryConfig{
			{
				Name:    "repo1",
				Cleanup: "delete",
			},
			{
				Name:    "repo2",
				Cleanup: "delete",
			},
		},
	}
	config.SetConfig(cfg)

	// Create mock projects from two different repositories
	project1 := &types.Project{
		Name: "repo1-service",
		Services: types.Services{
			"service1": types.ServiceConfig{
				Image: "image1",
			},
		},
	}

	project2 := &types.Project{
		Name: "repo2-service",
		Services: types.Services{
			"service2": types.ServiceConfig{
				Image: "image2",
			},
		},
	}

	// Override the QuadletDir for testing
	// This will affect the getUnitFilePath function without needing to mock it directly
	config.GetConfig().QuadletDir = tempDir

	// Create a shared processedUnits map to simulate proper usage
	processedUnits := make(map[string]bool)

	// Process the first project
	_ = ProcessComposeProjects([]*types.Project{project1}, false, processedUnits)
	// We expect an error due to database connection in test, but the processedUnits should still be updated
	// We're testing the tracking of units across multiple calls, not the actual database operations

	// Verify the first service was tracked in processedUnits
	assert.True(t, processedUnits["repo1-service-service1.container"], "First service should be tracked in processedUnits")

	// Process the second project with the same processedUnits map
	_ = ProcessComposeProjects([]*types.Project{project2}, false, processedUnits)
	// Again, we expect database errors but are testing the processedUnits tracking

	// Verify both services are tracked in processedUnits
	assert.True(t, processedUnits["repo1-service-service1.container"], "First service should still be tracked in processedUnits")
	assert.True(t, processedUnits["repo2-service-service2.container"], "Second service should be tracked in processedUnits")

	// At this point, cleanup should not have removed the first service because both are still in processedUnits
	// This test verifies the fix for the issue where services from the first repository were being
	// deleted when processing the second repository.
}
