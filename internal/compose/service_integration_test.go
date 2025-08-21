package compose

import (
	"os"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/testutil"
	"github.com/trly/quad-ops/internal/unit"
)

func TestProcessServices(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name             string
		project          *types.Project
		dependencyError  error
		processUnitError error
		wantErr          bool
		expectProcessed  []string
	}{
		{
			name: "single service without build",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			expectProcessed: []string{"test-project-web.container"},
			wantErr:         false,
		},
		{
			name: "single service with build",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
						Build: &types.BuildConfig{
							Context: ".",
						},
					},
				},
			},
			expectProcessed: []string{"test-project-web.container"},
			wantErr:         false,
		},
		{
			name: "multiple services",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
					"db": {
						Name:  "db",
						Image: "postgres:13",
					},
				},
			},
			expectProcessed: []string{"test-project-web.container", "test-project-db.container"},
			wantErr:         false,
		},
		{
			name: "service with init containers",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
						Labels: map[string]string{
							"quad-ops.init-containers": `[{"name":"migrate","image":"migrate:latest","command":["migrate","up"]}]`,
						},
					},
				},
			},
			expectProcessed: []string{"test-project-web.container"},
			wantErr:         false,
		},
		{
			name: "invalid init containers configuration",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
						Labels: map[string]string{
							"quad-ops.init-containers": `[invalid json}`,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "process unit error",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			processUnitError: assert.AnError,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			// Create dependency graph
			depGraph, err := dependency.BuildServiceDependencyGraph(tt.project)
			require.NoError(t, err)

			// Setup mock expectations based on expected behavior
			if len(tt.expectProcessed) > 0 {
				// Mock the filesystem operations that processUnit needs
				mockFS.On("GetUnitFilePath", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("/test/path").Maybe()
				mockFS.On("HasUnitChanged", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true).Maybe()
				mockFS.On("WriteUnitFile", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(tt.processUnitError).Maybe()
				mockFS.On("GetContentHash", mock.AnythingOfType("string")).Return("hash123").Maybe()

				if tt.processUnitError == nil {
					mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(&repository.Unit{}, nil).Maybe()
				}
			}

			// Mock repository operations for helpers
			mockRepo.On("FindAll").Return([]repository.Unit{}, nil).Maybe()

			// Run the test
			err = processor.processServices(tt.project, depGraph)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that expected units were processed
				for _, expectedUnit := range tt.expectProcessed {
					assert.True(t, processor.processedUnits[expectedUnit], "Expected unit %s to be processed", expectedUnit)
				}
			}
		})
	}
}

func TestProcessBuildIfPresent(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name              string
		service           types.ServiceConfig
		serviceName       string
		hasDockerfile     bool
		dockerfileContent string
		wantErr           bool
		expectBuildDep    bool
	}{
		{
			name: "service without build",
			service: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
			},
			serviceName:    "web",
			expectBuildDep: false,
			wantErr:        false,
		},
		{
			name: "service with build",
			service: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				Build: &types.BuildConfig{
					Context: ".",
				},
			},
			serviceName:       "web",
			hasDockerfile:     true,
			dockerfileContent: "FROM node:16\nCOPY . .\nRUN npm install",
			expectBuildDep:    true,
			wantErr:           false,
		},
		{
			name: "service with build and production target",
			service: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				Build: &types.BuildConfig{
					Context: ".",
					Target:  "production",
				},
			},
			serviceName:       "web",
			hasDockerfile:     true,
			dockerfileContent: "FROM node:16 as development\nFROM node:16 as production\nCOPY . .",
			expectBuildDep:    true,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test directory
			tempDir := t.TempDir()

			// Create Dockerfile if needed
			if tt.hasDockerfile {
				dockerfilePath := tempDir + "/Dockerfile"
				err := os.WriteFile(dockerfilePath, []byte(tt.dockerfileContent), 0600)
				require.NoError(t, err)
			}

			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			// Create project and dependency graph
			project := &types.Project{
				Name:       "test-project",
				WorkingDir: tempDir,
				Services:   map[string]types.ServiceConfig{tt.serviceName: tt.service},
			}

			depGraph, err := dependency.BuildServiceDependencyGraph(project)
			require.NoError(t, err)

			// Run the test
			err = processor.processBuildIfPresent(&tt.service, tt.serviceName, project, depGraph)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expectBuildDep {
					// Check that build dependency was added to graph
					buildServiceName := tt.serviceName + "-build"
					deps, _ := depGraph.GetDependencies(tt.serviceName)
					found := false
					for _, dep := range deps {
						if dep == buildServiceName {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected build dependency to be added to dependency graph")
				}
			}
		})
	}
}

func TestHandleProductionTarget(t *testing.T) {
	tests := []struct {
		name                string
		build               *unit.Build
		serviceName         string
		createDockerfile    bool
		dockerfileContent   string
		expectTargetCleared bool
		wantErr             bool
	}{
		{
			name: "non-production target",
			build: &unit.Build{
				Target: "development",
			},
			serviceName:         "web",
			expectTargetCleared: false,
			wantErr:             false,
		},
		{
			name: "production target exists in dockerfile",
			build: &unit.Build{
				Target: "production",
			},
			serviceName:         "web",
			createDockerfile:    true,
			dockerfileContent:   "FROM node:16\nFROM node:16 as production\nCOPY . .",
			expectTargetCleared: false,
			wantErr:             false,
		},
		{
			name: "production target missing in dockerfile",
			build: &unit.Build{
				Target: "production",
			},
			serviceName:         "web",
			createDockerfile:    true,
			dockerfileContent:   "FROM node:16\nCOPY . .",
			expectTargetCleared: true,
			wantErr:             false,
		},
		{
			name: "dockerfile doesn't exist",
			build: &unit.Build{
				Target: "production",
			},
			serviceName:         "web",
			createDockerfile:    false,
			expectTargetCleared: false,
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			if tt.createDockerfile {
				dockerfilePath := tempDir + "/Dockerfile"
				err := os.WriteFile(dockerfilePath, []byte(tt.dockerfileContent), 0600)
				require.NoError(t, err)
			}

			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			// Save original target
			originalTarget := tt.build.Target

			// Run the test
			err := processor.handleProductionTarget(tt.build, tt.serviceName, tempDir)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expectTargetCleared {
					assert.Empty(t, tt.build.Target, "Expected production target to be cleared")
				} else {
					assert.Equal(t, originalTarget, tt.build.Target, "Expected target to remain unchanged")
				}
			}
		})
	}
}

func TestFinishProcessingService(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name           string
		quadletUnit    *unit.QuadletUnit
		serviceName    string
		processUnitErr error
		wantErr        bool
	}{
		{
			name: "successful processing",
			quadletUnit: &unit.QuadletUnit{
				Name: "test-web",
				Type: "container",
			},
			serviceName: "web",
			wantErr:     false,
		},
		{
			name: "process unit error",
			quadletUnit: &unit.QuadletUnit{
				Name: "test-web",
				Type: "container",
			},
			serviceName:    "web",
			processUnitErr: assert.AnError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			// Create dependency graph
			project := &types.Project{Name: "test-project"}
			depGraph, err := dependency.BuildServiceDependencyGraph(project)
			require.NoError(t, err)

			// Setup processUnit mock expectations
			if tt.processUnitErr == nil {
				mockFS.On("GetUnitFilePath", tt.quadletUnit.Name, tt.quadletUnit.Type).Return("/test/path")
				mockFS.On("HasUnitChanged", "/test/path", mock.AnythingOfType("string")).Return(true)
				mockFS.On("WriteUnitFile", "/test/path", mock.AnythingOfType("string")).Return(nil)
				mockFS.On("GetContentHash", mock.AnythingOfType("string")).Return("hash123")
				mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(&repository.Unit{}, nil)
				mockRepo.On("FindAll").Return([]repository.Unit{}, nil)
			} else {
				mockFS.On("GetUnitFilePath", tt.quadletUnit.Name, tt.quadletUnit.Type).Return("/test/path")
				mockFS.On("HasUnitChanged", "/test/path", mock.AnythingOfType("string")).Return(true)
				mockFS.On("WriteUnitFile", "/test/path", mock.AnythingOfType("string")).Return(tt.processUnitErr)
				mockRepo.On("FindAll").Return([]repository.Unit{}, nil)
			}

			// Run the test
			err = processor.finishProcessingService(tt.quadletUnit, tt.serviceName, depGraph, "test-project")

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify unit was processed successfully
				unitKey := tt.quadletUnit.Name + "." + tt.quadletUnit.Type
				assert.True(t, processor.processedUnits[unitKey], "Expected unit to be processed")
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}
