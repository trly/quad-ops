package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestNewDefaultProcessor(t *testing.T) {
	processor := NewDefaultProcessor(true)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.repo)
	assert.NotNil(t, processor.systemd)
	assert.NotNil(t, processor.fs)
	assert.NotNil(t, processor.logger)
	assert.True(t, processor.force)
	assert.NotNil(t, processor.processedUnits)
	assert.NotNil(t, processor.changedUnits)
	assert.NotNil(t, processor.dependencyGraphs)
}

func TestProcessProjects(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name     string
		projects []*types.Project
		cleanup  bool
		wantErr  bool
	}{
		{
			name:     "empty projects list",
			projects: []*types.Project{},
			cleanup:  false,
			wantErr:  false,
		},
		{
			name: "single project with no services",
			projects: []*types.Project{
				{
					Name: "test-project",
				},
			},
			cleanup: false,
			wantErr: false,
		},
		{
			name: "single project with services",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {
							Name:  "web",
							Image: "nginx:latest",
						},
					},
				},
			},
			cleanup: false,
			wantErr: false,
		},
		{
			name: "multiple projects",
			projects: []*types.Project{
				{
					Name: "project1",
					Services: map[string]types.ServiceConfig{
						"web": {
							Name:  "web",
							Image: "nginx:latest",
						},
					},
				},
				{
					Name: "project2",
					Services: map[string]types.ServiceConfig{
						"db": {
							Name:  "db",
							Image: "postgres:13",
						},
					},
				},
			},
			cleanup: false,
			wantErr: false,
		},
		{
			name: "with cleanup enabled",
			projects: []*types.Project{
				{
					Name: "test-project",
					Services: map[string]types.ServiceConfig{
						"web": {
							Name:  "web",
							Image: "nginx:latest",
						},
					},
				},
			},
			cleanup: true,
			wantErr: false,
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

			err := processor.ProcessProjects(tt.projects, tt.cleanup)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithExistingProcessedUnits(t *testing.T) {
	// Setup mocks
	mockRepo := &MockRepository{}
	mockSystemd := &MockSystemdManager{}
	mockFS := &MockFileSystem{}
	logger := testutil.NewTestLogger(t)

	processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

	// Test with nil units
	result := processor.WithExistingProcessedUnits(nil)
	assert.Equal(t, processor, result)
	assert.Empty(t, processor.processedUnits)

	// Test with existing units
	existingUnits := map[string]bool{
		"web.container": true,
		"db.container":  true,
	}

	result = processor.WithExistingProcessedUnits(existingUnits)
	assert.Equal(t, processor, result)
	assert.Equal(t, existingUnits, processor.processedUnits)
}

func TestGetProcessedUnits(t *testing.T) {
	// Setup mocks
	mockRepo := &MockRepository{}
	mockSystemd := &MockSystemdManager{}
	mockFS := &MockFileSystem{}
	logger := testutil.NewTestLogger(t)

	processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

	// Initially empty
	units := processor.GetProcessedUnits()
	assert.Empty(t, units)

	// Add some units
	processor.processedUnits["web.container"] = true
	processor.processedUnits["db.container"] = true

	units = processor.GetProcessedUnits()
	expected := map[string]bool{
		"web.container": true,
		"db.container":  true,
	}
	assert.Equal(t, expected, units)
}
