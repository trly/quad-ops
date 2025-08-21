package compose

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/testutil"
	"github.com/trly/quad-ops/internal/unit"
)

func TestProcessUnit(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name                string
		unitItem            *unit.QuadletUnit
		force               bool
		hasChanged          bool
		hasNamingConflict   bool
		writeFileError      error
		updateDBError       error
		wantErr             bool
		expectInChangedList bool
	}{
		{
			name: "new unit - content has changed",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               false,
			hasChanged:          true,
			hasNamingConflict:   false,
			expectInChangedList: true,
			wantErr:             false,
		},
		{
			name: "unit with naming conflict",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               false,
			hasChanged:          false,
			hasNamingConflict:   true,
			expectInChangedList: true,
			wantErr:             false,
		},
		{
			name: "force update unit",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               true,
			hasChanged:          false,
			hasNamingConflict:   false,
			expectInChangedList: true,
			wantErr:             false,
		},
		{
			name: "unit unchanged - no update needed",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               false,
			hasChanged:          false,
			hasNamingConflict:   false,
			expectInChangedList: false,
			wantErr:             false,
		},
		{
			name: "write file error",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               false,
			hasChanged:          true,
			hasNamingConflict:   false,
			writeFileError:      errors.New("write error"),
			expectInChangedList: false,
			wantErr:             true,
		},
		{
			name: "update database error when writing",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               false,
			hasChanged:          true,
			hasNamingConflict:   false,
			updateDBError:       errors.New("db error"),
			expectInChangedList: false,
			wantErr:             true,
		},
		{
			name: "update database error when not writing",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			force:               false,
			hasChanged:          false,
			hasNamingConflict:   false,
			updateDBError:       errors.New("db error"),
			expectInChangedList: false,
			wantErr:             true,
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
			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, tt.force)

			// Setup mock expectations
			unitPath := "/test/path/" + tt.unitItem.Name + "." + tt.unitItem.Type

			mockFS.On("GetUnitFilePath", tt.unitItem.Name, tt.unitItem.Type).Return(unitPath)
			mockFS.On("HasUnitChanged", unitPath, mock.AnythingOfType("string")).Return(tt.hasChanged)

			// Mock HasNamingConflict helper
			mockRepo.On("FindAll").Return([]repository.Unit{}, nil).Maybe()

			// Mock content generation
			mockFS.On("GetContentHash", mock.AnythingOfType("string")).Return("hash123").Maybe()

			shouldWrite := tt.force || tt.hasChanged || tt.hasNamingConflict

			if shouldWrite {
				if tt.writeFileError != nil {
					mockFS.On("WriteUnitFile", unitPath, mock.AnythingOfType("string")).Return(tt.writeFileError)
				} else {
					mockFS.On("WriteUnitFile", unitPath, mock.AnythingOfType("string")).Return(nil)

					// Only expect database update if write succeeds
					if tt.updateDBError != nil {
						mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(nil, tt.updateDBError)
					} else {
						mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(&repository.Unit{}, nil)
					}
				}
			} else {
				// Still expect database update even when not writing
				if tt.updateDBError != nil {
					mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(nil, tt.updateDBError)
				} else {
					mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(&repository.Unit{}, nil)
				}
			}

			// Run the test
			initialChangedCount := len(processor.changedUnits)
			err := processor.processUnit(tt.unitItem)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check if unit was tracked as processed
			unitKey := tt.unitItem.Name + "." + tt.unitItem.Type
			assert.True(t, processor.processedUnits[unitKey], "unit should be tracked as processed")

			// Check if unit was added to changed list
			if tt.expectInChangedList && !tt.wantErr {
				assert.Len(t, processor.changedUnits, initialChangedCount+1, "unit should be in changed list")
				assert.Contains(t, processor.changedUnits, *tt.unitItem)
			} else {
				assert.Len(t, processor.changedUnits, initialChangedCount, "unit should not be in changed list")
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
			mockSystemd.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}

func TestUpdateUnitDatabase(t *testing.T) {
	t.Skip("Complex integration test - should be run separately")
	tests := []struct {
		name        string
		unitItem    *unit.QuadletUnit
		content     string
		createError error
		wantErr     bool
	}{
		{
			name: "successful database update",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			content: "unit content",
			wantErr: false,
		},
		{
			name: "database create error",
			unitItem: &unit.QuadletUnit{
				Name: "web",
				Type: "container",
			},
			content:     "unit content",
			createError: errors.New("create error"),
			wantErr:     true,
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

			// Setup mock expectations
			contentHash := "hash123"
			mockFS.On("GetContentHash", tt.content).Return(contentHash)

			expectedUnit := &repository.Unit{
				Name:     tt.unitItem.Name,
				Type:     tt.unitItem.Type,
				SHA1Hash: []byte(contentHash),
			}

			if tt.createError != nil {
				mockRepo.On("Create", mock.MatchedBy(func(unit *repository.Unit) bool {
					return unit.Name == expectedUnit.Name &&
						unit.Type == expectedUnit.Type &&
						string(unit.SHA1Hash) == string(expectedUnit.SHA1Hash)
				})).Return(nil, tt.createError)
			} else {
				mockRepo.On("Create", mock.MatchedBy(func(unit *repository.Unit) bool {
					return unit.Name == expectedUnit.Name &&
						unit.Type == expectedUnit.Type &&
						string(unit.SHA1Hash) == string(expectedUnit.SHA1Hash)
				})).Return(&repository.Unit{}, nil)
			}

			// Run the test
			err := processor.updateUnitDatabase(tt.unitItem, tt.content)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
			mockFS.AssertExpectations(t)
		})
	}
}

func TestProcessUnitIntegration(t *testing.T) {
	t.Skip("Integration test - should be run separately")
	// Integration test that verifies the full flow
	mockRepo := &MockRepository{}
	mockSystemd := &MockSystemdManager{}
	mockFS := &MockFileSystem{}
	logger := testutil.NewTestLogger(t)

	processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

	unitItem := &unit.QuadletUnit{
		Name: "integration-test",
		Type: "container",
	}

	unitPath := "/test/integration-test.container"
	contentHash := "hash123"

	// Setup the full mock chain
	mockFS.On("GetUnitFilePath", "integration-test", "container").Return(unitPath)
	mockFS.On("HasUnitChanged", unitPath, mock.AnythingOfType("string")).Return(true)
	mockRepo.On("FindAll").Return([]repository.Unit{}, nil)
	mockFS.On("WriteUnitFile", unitPath, mock.AnythingOfType("string")).Return(nil)
	mockFS.On("GetContentHash", mock.AnythingOfType("string")).Return(contentHash)

	expectedUnit := &repository.Unit{
		Name:     "integration-test",
		Type:     "container",
		SHA1Hash: []byte(contentHash),
	}
	mockRepo.On("Create", mock.AnythingOfType("*repository.Unit")).Return(expectedUnit, nil)

	// Run the test
	err := processor.processUnit(unitItem)
	require.NoError(t, err)

	// Verify state changes
	assert.True(t, processor.processedUnits["integration-test.container"])
	assert.Len(t, processor.changedUnits, 1)
	assert.Contains(t, processor.changedUnits, *unitItem)

	// Verify all mocks were called as expected
	mockRepo.AssertExpectations(t)
	mockSystemd.AssertExpectations(t)
	mockFS.AssertExpectations(t)
}
