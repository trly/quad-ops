package compose

import (
	"log/slog"

	"github.com/stretchr/testify/mock"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
)

// MockRepository is a mock implementation of the Repository interface.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindAll() ([]repository.Unit, error) {
	args := m.Called()
	return args.Get(0).([]repository.Unit), args.Error(1)
}

func (m *MockRepository) Create(unit *repository.Unit) (*repository.Unit, error) {
	args := m.Called(unit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Unit), args.Error(1)
}

func (m *MockRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockSystemdManager is a mock implementation of the SystemdManager interface.
type MockSystemdManager struct {
	mock.Mock
}

func (m *MockSystemdManager) RestartChangedUnits(units []systemd.UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	args := m.Called(units, projectDependencyGraphs)
	return args.Error(0)
}

func (m *MockSystemdManager) ReloadSystemd() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSystemdManager) StopUnit(name, unitType string) error {
	args := m.Called(name, unitType)
	return args.Error(0)
}

// MockFileSystem is a mock implementation of the FileSystem interface.
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) GetUnitFilePath(name, unitType string) string {
	args := m.Called(name, unitType)
	return args.String(0)
}

func (m *MockFileSystem) HasUnitChanged(unitPath, content string) bool {
	args := m.Called(unitPath, content)
	return args.Bool(0)
}

func (m *MockFileSystem) WriteUnitFile(unitPath, content string) error {
	args := m.Called(unitPath, content)
	return args.Error(0)
}

func (m *MockFileSystem) GetContentHash(content string) string {
	args := m.Called(content)
	return args.String(0)
}

// initTestLogger initializes a test logger.
func initTestLogger() *slog.Logger {
	log.Init(true)
	return log.GetLogger()
}
