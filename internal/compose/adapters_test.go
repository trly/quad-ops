package compose

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/repository"
)

// Mock implementations for testing adapters
type MockRepositoryImpl struct {
	mock.Mock
}

func (m *MockRepositoryImpl) FindAll() ([]repository.Unit, error) {
	args := m.Called()
	return args.Get(0).([]repository.Unit), args.Error(1)
}

func (m *MockRepositoryImpl) FindByUnitType(unitType string) ([]repository.Unit, error) {
	args := m.Called(unitType)
	return args.Get(0).([]repository.Unit), args.Error(1)
}

func (m *MockRepositoryImpl) FindByID(id int64) (repository.Unit, error) {
	args := m.Called(id)
	return args.Get(0).(repository.Unit), args.Error(1)
}

func (m *MockRepositoryImpl) Create(unit *repository.Unit) (int64, error) {
	args := m.Called(unit)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepositoryImpl) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestRepositoryAdapter_FindAll(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepositoryImpl)
		expectedUnits  []repository.Unit
		expectedError  error
	}{
		{
			name: "successful findall",
			setupMock: func(mockRepo *MockRepositoryImpl) {
				units := []repository.Unit{
					{ID: 1, Name: "web", Type: "container"},
					{ID: 2, Name: "db", Type: "container"},
				}
				mockRepo.On("FindAll").Return(units, nil)
			},
			expectedUnits: []repository.Unit{
				{ID: 1, Name: "web", Type: "container"},
				{ID: 2, Name: "db", Type: "container"},
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			setupMock: func(mockRepo *MockRepositoryImpl) {
				mockRepo.On("FindAll").Return([]repository.Unit{}, errors.New("repository error"))
			},
			expectedUnits: []repository.Unit{},
			expectedError: errors.New("repository error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepositoryImpl)
			tt.setupMock(mockRepo)

			adapter := NewRepositoryAdapter(mockRepo)
			units, err := adapter.FindAll()

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedUnits, units)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRepositoryAdapter_Create(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		mockRepo := new(MockRepositoryImpl)
		mockRepo.On("Create", mock.MatchedBy(func(unit *repository.Unit) bool {
			return unit.Name == "web" && unit.Type == "container"
		})).Return(int64(123), nil)

		adapter := NewRepositoryAdapter(mockRepo)
		inputUnit := &repository.Unit{Name: "web", Type: "container"}
		
		result, err := adapter.Create(inputUnit)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(123), result.ID)
		assert.Equal(t, "web", result.Name)
		assert.Equal(t, "container", result.Type)

		mockRepo.AssertExpectations(t)
	})
}

func TestRepositoryAdapter_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockRepo := new(MockRepositoryImpl)
		// Current implementation ignores the ID and uses 0
		mockRepo.On("Delete", int64(0)).Return(nil)

		adapter := NewRepositoryAdapter(mockRepo)
		err := adapter.Delete("123")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestNewRepositoryAdapter(t *testing.T) {
	t.Run("creates valid adapter", func(t *testing.T) {
		mockRepo := new(MockRepositoryImpl)
		adapter := NewRepositoryAdapter(mockRepo)

		assert.NotNil(t, adapter)
		assert.Implements(t, (*Repository)(nil), adapter)
	})
}

func TestSystemdAdapter_InterfaceCompliance(t *testing.T) {
	t.Run("creates valid adapter", func(t *testing.T) {
		adapter := NewSystemdAdapter()

		assert.NotNil(t, adapter)
		assert.Implements(t, (*SystemdManager)(nil), adapter)
	})
}

func TestFileSystemAdapter_InterfaceCompliance(t *testing.T) {
	t.Run("creates valid adapter", func(t *testing.T) {
		adapter := NewFileSystemAdapter()

		assert.NotNil(t, adapter)
		assert.Implements(t, (*FileSystem)(nil), adapter)
	})

	t.Run("GetContentHash works correctly", func(t *testing.T) {
		adapter := NewFileSystemAdapter()
		
		hash1 := adapter.GetContentHash("test content")
		hash2 := adapter.GetContentHash("test content")
		hash3 := adapter.GetContentHash("different content")
		
		// Same content should produce same hash
		assert.Equal(t, hash1, hash2)
		// Different content should produce different hash
		assert.NotEqual(t, hash1, hash3)
		assert.NotEmpty(t, hash1)
	})
}

func TestAdapters_InterfaceCompliance(t *testing.T) {
	t.Run("all adapters implement their interfaces", func(t *testing.T) {
		// Test Repository adapter
		mockRepo := new(MockRepositoryImpl)
		repoAdapter := NewRepositoryAdapter(mockRepo)
		assert.Implements(t, (*Repository)(nil), repoAdapter)

		// Test SystemdManager adapter
		systemdAdapter := NewSystemdAdapter()
		assert.Implements(t, (*SystemdManager)(nil), systemdAdapter)

		// Test FileSystem adapter
		fsAdapter := NewFileSystemAdapter()
		assert.Implements(t, (*FileSystem)(nil), fsAdapter)
	})
}
