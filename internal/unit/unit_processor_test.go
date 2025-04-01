package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
)

// MockUnitRepository is a mock implementation of UnitRepository
type MockUnitRepository struct {
	mock.Mock
}

func (m *MockUnitRepository) FindAll() ([]Unit, error) {
	args := m.Called()
	return args.Get(0).([]Unit), args.Error(1)
}

func (m *MockUnitRepository) FindByUnitType(unitType string) ([]Unit, error) {
	args := m.Called(unitType)
	return args.Get(0).([]Unit), args.Error(1)
}

func (m *MockUnitRepository) FindByID(id int64) (Unit, error) {
	args := m.Called(id)
	return args.Get(0).(Unit), args.Error(1)
}

func (m *MockUnitRepository) Create(unit *Unit) (int64, error) {
	args := m.Called(unit)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUnitRepository) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockSystemdUnit is a mock implementation of SystemdUnit
type MockSystemdUnit struct {
	mock.Mock
}

func (m *MockSystemdUnit) GetServiceName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSystemdUnit) GetUnitType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSystemdUnit) GetUnitName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSystemdUnit) GetStatus() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSystemdUnit) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSystemdUnit) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSystemdUnit) Restart() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSystemdUnit) Show() error {
	args := m.Called()
	return args.Error(0)
}

// setupTemporaryDir creates a temporary directory for testing
func setupTemporaryDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "quad-ops-test")
	assert.NoError(t, err)
	return dir
}

// cleanupTemporaryDir removes the temporary directory
func cleanupTemporaryDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	assert.NoError(t, err)
}

func TestUpdateUnitDatabase(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := setupTemporaryDir(t)
	defer cleanupTemporaryDir(t, tempDir)

	// Set up config before creating repos
	cfg := &config.Config{
		RepositoryDir: tempDir,
		QuadletDir:    filepath.Join(tempDir, "quadlet"),
	}
	config.SetConfig(cfg)

	tests := []struct {
		name             string
		repositoryConfig config.RepositoryConfig
		content          string
		unit             QuadletUnit
		expectCleanup    string
	}{
		{
			name: "Default Cleanup Policy",
			repositoryConfig: config.RepositoryConfig{
				Name:      "test-repo",
				URL:       "https://example.com/repo.git",
				Reference: "main",
				Cleanup:   "", // Default is empty which should result in "keep"
			},
			content: "test content",
			unit: QuadletUnit{
				Name: "test-unit",
				Type: "container",
			},
			expectCleanup: "keep",
		},
		{
			name: "Explicit Keep Cleanup Policy",
			repositoryConfig: config.RepositoryConfig{
				Name:      "test-repo",
				URL:       "https://example.com/repo.git",
				Reference: "main",
				Cleanup:   "keep",
			},
			content: "test content",
			unit: QuadletUnit{
				Name: "test-unit",
				Type: "container",
			},
			expectCleanup: "keep",
		},
		{
			name: "Delete Cleanup Policy",
			repositoryConfig: config.RepositoryConfig{
				Name:      "test-repo",
				URL:       "https://example.com/repo.git",
				Reference: "main",
				Cleanup:   "delete",
			},
			content: "test content",
			unit: QuadletUnit{
				Name: "test-unit",
				Type: "container",
			},
			expectCleanup: "delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := new(MockUnitRepository)
			mockRepo.On("Create", mock.MatchedBy(func(unit *Unit) bool {
				return unit.Name == tt.unit.Name &&
					unit.Type == tt.unit.Type &&
					unit.CleanupPolicy == tt.expectCleanup
			})).Return(int64(1), nil)

			// Create a git repository directly without using the NewGitRepository function
			gitRepo := &git.GitRepository{
				RepositoryConfig: tt.repositoryConfig,
				Path:             filepath.Join(tempDir, tt.repositoryConfig.Name),
			}

			// Create processor
			processor := &UnitProcessor{
				repo:     gitRepo,
				unitRepo: mockRepo,
				verbose:  false,
			}

			// Call method to test
			err := processor.updateUnitDatabase(&tt.unit, tt.content)
			assert.NoError(t, err)

			// Verify the mock was called with expected arguments
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestCleanupProcessor is a test-specific processor implementation
type TestCleanupProcessor struct {
	UnitProcessor
	StopCalled   bool
	ReloadCalled bool
}

// Override methods that call system functions
func (t *TestCleanupProcessor) cleanupOrphanedUnits(processedUnits map[string]bool) error {
	dbUnits, err := t.unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)
		if !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete") {
			// Record that we would have called stop
			t.StopCalled = true

			// Remove the unit file
			unitPath := filepath.Join(config.GetConfig().QuadletDir, unitKey)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					fmt.Printf("error removing orphaned unit file %s: %v\n", unitPath, err)
				}
			}

			// Remove from database
			if err := t.unitRepo.Delete(dbUnit.ID); err != nil {
				fmt.Printf("error deleting unit %s from database: %v\n", unitKey, err)
				continue
			}
		}
	}

	// Record that we would have called reload
	t.ReloadCalled = true

	return nil
}

func TestCleanupOrphanedUnits(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := setupTemporaryDir(t)
	defer cleanupTemporaryDir(t, tempDir)

	// Set up config
	cfg := &config.Config{
		RepositoryDir: tempDir,
		QuadletDir:    tempDir,
	}
	config.SetConfig(cfg)

	// Create test units
	testUnits := []Unit{
		{
			ID:            1,
			Name:          "unit1",
			Type:          "container",
			CleanupPolicy: "keep",
			CreatedAt:     time.Now(),
		},
		{
			ID:            2,
			Name:          "unit2",
			Type:          "container",
			CleanupPolicy: "delete",
			CreatedAt:     time.Now(),
		},
		{
			ID:            3,
			Name:          "unit3",
			Type:          "volume",
			CleanupPolicy: "delete",
			CreatedAt:     time.Now(),
		},
	}

	// Create test files
	for _, unit := range testUnits {
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
		filePath := filepath.Join(tempDir, unitKey)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		assert.NoError(t, err)
	}

	// Create processed units map
	processedUnits := map[string]bool{
		"unit1.container": true, // This unit is in the processed map, should not be cleaned up
		// unit2.container and unit3.volume are not in the processed map and should be cleaned up
	}

	// Setup mock repository
	mockRepo := new(MockUnitRepository)
	mockRepo.On("FindAll").Return(testUnits, nil)
	mockRepo.On("Delete", int64(2)).Return(nil) // unit2 should be deleted
	mockRepo.On("Delete", int64(3)).Return(nil) // unit3 should be deleted

	// Setup repository
	gitRepo := &git.GitRepository{
		RepositoryConfig: config.RepositoryConfig{
			Name: "test-repo",
			URL:  "https://example.com/repo.git",
		},
		Path: filepath.Join(tempDir, "test-repo"),
	}

	// Create test processor
	processor := &TestCleanupProcessor{
		UnitProcessor: UnitProcessor{
			repo:     gitRepo,
			unitRepo: mockRepo,
			verbose:  false,
		},
		StopCalled:   false,
		ReloadCalled: false,
	}

	// Call the method under test
	err := processor.cleanupOrphanedUnits(processedUnits)
	assert.NoError(t, err)

	// Verify our test functions were called
	assert.True(t, processor.StopCalled, "Stop should have been called")
	assert.True(t, processor.ReloadCalled, "Reload should have been called")

	// Verify the mock was called as expected
	mockRepo.AssertExpectations(t)

	// Verify files that should be deleted are gone, and files that should remain are still there
	_, err = os.Stat(filepath.Join(tempDir, "unit1.container"))
	assert.NoError(t, err, "unit1.container should still exist")

	_, err = os.Stat(filepath.Join(tempDir, "unit2.container"))
	assert.True(t, os.IsNotExist(err), "unit2.container should have been deleted")

	_, err = os.Stat(filepath.Join(tempDir, "unit3.volume"))
	assert.True(t, os.IsNotExist(err), "unit3.volume should have been deleted")
}
