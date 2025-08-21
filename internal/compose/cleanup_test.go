package compose

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestCleanupOrphans(t *testing.T) {
	tests := []struct {
		name             string
		existingUnits    []repository.Unit
		processedUnits   map[string]bool
		repoFindAllErr   error
		repoDeleteErr    error
		systemdStopErr   error
		systemdReloadErr error
		fsRemoveErr      error
		wantError        bool
	}{
		{
			name: "no orphaned units",
			existingUnits: []repository.Unit{
				{ID: 1, Name: "web", Type: "container"},
				{ID: 2, Name: "db", Type: "container"},
			},
			processedUnits: map[string]bool{
				"web.container": true,
				"db.container":  true,
			},
			wantError: false,
		},
		{
			name: "single orphaned unit cleanup success",
			existingUnits: []repository.Unit{
				{ID: 1, Name: "web", Type: "container"},
				{ID: 2, Name: "orphaned", Type: "container"},
			},
			processedUnits: map[string]bool{
				"web.container": true,
			},
			wantError: false,
		},
		{
			name: "multiple orphaned units",
			existingUnits: []repository.Unit{
				{ID: 1, Name: "web", Type: "container"},
				{ID: 2, Name: "orphaned1", Type: "container"},
				{ID: 3, Name: "orphaned2", Type: "volume"},
			},
			processedUnits: map[string]bool{
				"web.container": true,
			},
			wantError: false,
		},
		{
			name:           "repository findall error",
			repoFindAllErr: errors.New("repository error"),
			wantError:      true,
		},
		{
			name: "systemd stop error - continues with cleanup",
			existingUnits: []repository.Unit{
				{ID: 1, Name: "orphaned", Type: "container"},
			},
			processedUnits: map[string]bool{},
			systemdStopErr: errors.New("stop error"),
			wantError:      false,
		},
		{
			name: "repository delete error - continues with next unit",
			existingUnits: []repository.Unit{
				{ID: 1, Name: "orphaned1", Type: "container"},
				{ID: 2, Name: "orphaned2", Type: "container"},
			},
			processedUnits: map[string]bool{},
			repoDeleteErr:  errors.New("delete error"),
			wantError:      false,
		},
		{
			name: "systemd reload error - logs but doesn't fail",
			existingUnits: []repository.Unit{
				{ID: 1, Name: "orphaned", Type: "container"},
			},
			processedUnits:   map[string]bool{},
			systemdReloadErr: errors.New("reload error"),
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory for unit files
			tempDir := t.TempDir()

			// Create test unit files that will be removed
			for _, unit := range tt.existingUnits {
				if _, found := tt.processedUnits[unit.Name+"."+unit.Type]; !found {
					unitPath := tempDir + "/" + unit.Name + "." + unit.Type
					err := os.WriteFile(unitPath, []byte("test content"), 0600)
					require.NoError(t, err)
				}
			}

			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			// Configure mock behaviors
			mockRepo.On("FindAll").Return(tt.existingUnits, tt.repoFindAllErr)

			// Only set up SystemdReloadSystemd if FindAll doesn't return an error
			if tt.repoFindAllErr == nil {
				mockSystemd.On("ReloadSystemd").Return(tt.systemdReloadErr)

				for _, unit := range tt.existingUnits {
					unitKey := unit.Name + "." + unit.Type
					if _, processed := tt.processedUnits[unitKey]; !processed {
						// This unit is orphaned, expect cleanup calls
						mockSystemd.On("StopUnit", unit.Name, unit.Type).Return(tt.systemdStopErr)

						unitPath := tempDir + "/" + unit.Name + "." + unit.Type
						mockFS.On("GetUnitFilePath", unit.Name, unit.Type).Return(unitPath)

						if tt.repoDeleteErr != nil {
							mockRepo.On("Delete", mock.AnythingOfType("string")).Return(tt.repoDeleteErr)
						} else {
							mockRepo.On("Delete", mock.AnythingOfType("string")).Return(nil)
						}
					}
				}
			}

			// Create processor
			p := &Processor{
				repo:           mockRepo,
				systemd:        mockSystemd,
				fs:             mockFS,
				logger:         logger,
				processedUnits: tt.processedUnits,
			}

			// Run cleanup
			err := p.cleanupOrphans()

			// Check results
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify unit files were removed for orphaned units
			for _, unit := range tt.existingUnits {
				unitKey := unit.Name + "." + unit.Type
				unitPath := tempDir + "/" + unit.Name + "." + unit.Type

				if _, processed := tt.processedUnits[unitKey]; !processed {
					// Orphaned unit should have file removed
					_, err := os.Stat(unitPath)
					assert.True(t, os.IsNotExist(err), "orphaned unit file should be removed")
				}
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
			mockSystemd.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}

func TestCleanupOrphansFileRemovalScenarios(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   bool
		expectError bool
	}{
		{
			name:        "file exists and gets removed",
			setupFile:   true,
			expectError: false,
		},
		{
			name:        "file doesn't exist - no error",
			setupFile:   false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			unitPath := tempDir + "/test.container"

			if tt.setupFile {
				err := os.WriteFile(unitPath, []byte("content"), 0600)
				require.NoError(t, err)
			}

			// Setup mocks
			mockRepo := &MockRepository{}
			mockSystemd := &MockSystemdManager{}
			mockFS := &MockFileSystem{}
			logger := testutil.NewTestLogger(t)

			existingUnit := repository.Unit{ID: 1, Name: "test", Type: "container"}

			mockRepo.On("FindAll").Return([]repository.Unit{existingUnit}, nil)
			mockSystemd.On("StopUnit", "test", "container").Return(nil)
			mockFS.On("GetUnitFilePath", "test", "container").Return(unitPath)
			mockRepo.On("Delete", "1").Return(nil)
			mockSystemd.On("ReloadSystemd").Return(nil)

			p := &Processor{
				repo:           mockRepo,
				systemd:        mockSystemd,
				fs:             mockFS,
				logger:         logger,
				processedUnits: map[string]bool{}, // No processed units = all are orphaned
			}

			err := p.cleanupOrphans()
			assert.NoError(t, err)

			// Verify file was removed if it existed
			_, err = os.Stat(unitPath)
			assert.True(t, os.IsNotExist(err))

			mockRepo.AssertExpectations(t)
			mockSystemd.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}
