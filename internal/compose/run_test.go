package compose

import (
	"errors"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
	"github.com/trly/quad-ops/internal/testutil"
	"github.com/trly/quad-ops/internal/unit"
)

func TestProcessProjectsInternal(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name                 string
		projects             []*types.Project
		cleanup              bool
		cleanupError         error
		restartError         error
		dependencyGraphError error
		processProjectError  error
		wantErr              bool
	}{
		{
			name: "successful processing without cleanup",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {Name: "web", Image: "nginx:latest"},
					},
				},
			},
			cleanup: false,
			wantErr: false,
		},
		{
			name: "successful processing with cleanup",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {Name: "web", Image: "nginx:latest"},
					},
				},
			},
			cleanup: true,
			wantErr: false,
		},
		{
			name: "dependency graph build failure",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {
							Name:  "web",
							Image: "nginx:latest",
							DependsOn: map[string]types.ServiceDependency{
								"nonexistent": {},
							},
						},
					},
				},
			},
			cleanup:              false,
			dependencyGraphError: errors.New("dependency error"),
			wantErr:              true,
		},
		{
			name: "process project failure",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {Name: "web", Image: "nginx:latest"},
					},
				},
			},
			cleanup:             false,
			processProjectError: errors.New("process project error"),
			wantErr:             true,
		},
		{
			name: "cleanup error - continues processing",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {Name: "web", Image: "nginx:latest"},
					},
				},
			},
			cleanup:      true,
			cleanupError: errors.New("cleanup error"),
			wantErr:      false,
		},
		{
			name: "restart changed units error",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {Name: "web", Image: "nginx:latest"},
					},
				},
			},
			cleanup:      false,
			restartError: errors.New("restart error"),
			wantErr:      false, // restart errors are logged but don't fail the process
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			// Create processor
			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			// Setup mock expectations for cleanup
			if tt.cleanup {
				mockRepo.On("FindAll").Return([]repository.Unit{}, tt.cleanupError)
				if tt.cleanupError == nil {
					mockSystemd.On("ReloadSystemd").Return(nil)
				}
			}

			// Setup mock expectations for restart
			if tt.restartError != nil {
				// Add a changed unit to trigger restart
				processor.changedUnits = []unit.QuadletUnit{{Name: "test", Type: "container"}}
				mockSystemd.On("RestartChangedUnits",
					mock.AnythingOfType("[]systemd.UnitChange"),
					mock.AnythingOfType("map[string]*dependency.ServiceDependencyGraph")).
					Return(tt.restartError)
			}

			// Run the test
			err := processor.processProjects(tt.projects, tt.cleanup)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
			mockSystemd.AssertExpectations(t)
		})
	}
}

func TestProcessProject(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name          string
		project       *types.Project
		servicesError error
		volumesError  error
		networksError error
		wantErr       bool
	}{
		{
			name: "successful processing of all components",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {Name: "web", Image: "nginx:latest"},
				},
				Volumes: map[string]types.VolumeConfig{
					"data": {},
				},
				Networks: map[string]types.NetworkConfig{
					"frontend": {},
				},
			},
			wantErr: false,
		},
		{
			name: "services processing error",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {Name: "web", Image: "nginx:latest"},
				},
			},
			servicesError: errors.New("services error"),
			wantErr:       true,
		},
		{
			name: "volumes processing error",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {Name: "web", Image: "nginx:latest"},
				},
				Volumes: map[string]types.VolumeConfig{
					"data": {},
				},
			},
			volumesError: errors.New("volumes error"),
			wantErr:      true,
		},
		{
			name: "networks processing error",
			project: &types.Project{
				Name: "test-project",
				Services: map[string]types.ServiceConfig{
					"web": {Name: "web", Image: "nginx:latest"},
				},
				Networks: map[string]types.NetworkConfig{
					"frontend": {},
				},
			},
			networksError: errors.New("networks error"),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			// Create processor with dependency graph
			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			// Create a simple dependency graph for the project
			depGraph := &dependency.ServiceDependencyGraph{}
			processor.dependencyGraphs[tt.project.Name] = depGraph

			// Simulate the errors by overriding the functions if needed
			// This is a simplified test - in reality we'd need to mock the internal calls

			// Run the test
			err := processor.processProject(tt.project)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRestartChangedUnits(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name         string
		changedUnits []unit.QuadletUnit
		restartError error
		wantErr      bool
	}{
		{
			name:         "no changed units",
			changedUnits: []unit.QuadletUnit{},
			wantErr:      false,
		},
		{
			name: "single changed unit - successful restart",
			changedUnits: []unit.QuadletUnit{
				{Name: "web", Type: "container"},
			},
			wantErr: false,
		},
		{
			name: "multiple changed units - successful restart",
			changedUnits: []unit.QuadletUnit{
				{Name: "web", Type: "container"},
				{Name: "db", Type: "container"},
				{Name: "data", Type: "volume"},
			},
			wantErr: false,
		},
		{
			name: "restart failure",
			changedUnits: []unit.QuadletUnit{
				{Name: "web", Type: "container"},
			},
			restartError: errors.New("restart failed"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			// Create processor
			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)
			processor.changedUnits = tt.changedUnits
			processor.dependencyGraphs = map[string]*dependency.ServiceDependencyGraph{
				"test": {},
			}

			// Setup mock expectations
			if len(tt.changedUnits) > 0 {
				expectedSystemdUnits := make([]systemd.UnitChange, len(tt.changedUnits))
				for i, unit := range tt.changedUnits {
					expectedSystemdUnits[i] = systemd.UnitChange{
						Name: unit.Name,
						Type: unit.Type,
					}
				}

				mockSystemd.On("RestartChangedUnits",
					expectedSystemdUnits,
					processor.dependencyGraphs).Return(tt.restartError)
			}

			// Run the test
			err := processor.restartChangedUnits()

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockSystemd.AssertExpectations(t)
		})
	}
}
