package cmd

import (
	"testing"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
	"github.com/trly/quad-ops/internal/testutil"
)

// MockValidator implements SystemValidator for testing.
type MockValidator struct {
	SystemRequirementsFunc func() error
}

func (m *MockValidator) SystemRequirements() error {
	if m.SystemRequirementsFunc != nil {
		return m.SystemRequirementsFunc()
	}
	return nil
}

// MockUnitRepo implements repository.Repository for testing.
type MockUnitRepo struct {
	FindAllFunc func() ([]repository.Unit, error)
}

func (m *MockUnitRepo) FindAll() ([]repository.Unit, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc()
	}
	return []repository.Unit{}, nil
}

func (m *MockUnitRepo) FindByUnitType(string) ([]repository.Unit, error) {
	return []repository.Unit{}, nil
}
func (m *MockUnitRepo) FindByID(int64) (repository.Unit, error) { return repository.Unit{}, nil }
func (m *MockUnitRepo) Create(*repository.Unit) (int64, error)  { return 0, nil }
func (m *MockUnitRepo) Delete(int64) error                      { return nil }

// MockUnitManager implements systemd.UnitManager for testing.
type MockUnitManager struct {
	StartFunc        func(string, string) error
	StopFunc         func(string, string) error
	ResetFailedFunc  func(string, string) error
	StartCalls       []StartCall
	StopCalls        []StopCall
	ResetFailedCalls []ResetFailedCall
}

type StartCall struct {
	Name, UnitType string
}

type StopCall struct {
	Name, UnitType string
}

type ResetFailedCall struct {
	Name, UnitType string
}

func (m *MockUnitManager) Start(name, unitType string) error {
	m.StartCalls = append(m.StartCalls, StartCall{Name: name, UnitType: unitType})
	if m.StartFunc != nil {
		return m.StartFunc(name, unitType)
	}
	return nil
}

func (m *MockUnitManager) Stop(name, unitType string) error {
	m.StopCalls = append(m.StopCalls, StopCall{Name: name, UnitType: unitType})
	if m.StopFunc != nil {
		return m.StopFunc(name, unitType)
	}
	return nil
}

func (m *MockUnitManager) ResetFailed(name, unitType string) error {
	m.ResetFailedCalls = append(m.ResetFailedCalls, ResetFailedCall{Name: name, UnitType: unitType})
	if m.ResetFailedFunc != nil {
		return m.ResetFailedFunc(name, unitType)
	}
	return nil
}

func (m *MockUnitManager) GetUnit(string, string) systemd.Unit      { return nil }
func (m *MockUnitManager) GetStatus(string, string) (string, error) { return "", nil }
func (m *MockUnitManager) Restart(string, string) error             { return nil }
func (m *MockUnitManager) Show(string, string) error                { return nil }
func (m *MockUnitManager) ReloadSystemd() error                     { return nil }
func (m *MockUnitManager) GetUnitFailureDetails(string) string      { return "" }

// AppBuilder provides a fluent interface for building test Apps.
type AppBuilder struct {
	logger      log.Logger
	config      *config.Settings
	validator   SystemValidator
	unitRepo    repository.Repository
	unitManager systemd.UnitManager
}

// NewAppBuilder creates a new AppBuilder with sensible defaults.
func NewAppBuilder(t *testing.T) *AppBuilder {
	return &AppBuilder{
		logger:      testutil.NewTestLogger(t),
		config:      &config.Settings{Verbose: false},
		validator:   &MockValidator{},
		unitRepo:    &MockUnitRepo{},
		unitManager: &MockUnitManager{},
	}
}

func (b *AppBuilder) WithValidator(v SystemValidator) *AppBuilder {
	b.validator = v
	return b
}

func (b *AppBuilder) WithUnitRepo(r repository.Repository) *AppBuilder {
	b.unitRepo = r
	return b
}

func (b *AppBuilder) WithUnitManager(m systemd.UnitManager) *AppBuilder {
	b.unitManager = m
	return b
}

func (b *AppBuilder) WithConfig(c *config.Settings) *AppBuilder {
	b.config = c
	return b
}

func (b *AppBuilder) WithVerbose(verbose bool) *AppBuilder {
	b.config.Verbose = verbose
	return b
}

func (b *AppBuilder) Build(t *testing.T) *App {
	return &App{
		Logger:         b.logger,
		Config:         b.config,
		ConfigProvider: testutil.NewMockConfig(t),
		Runner:         &execx.RealRunner{},
		FSService:      &fs.Service{},
		UnitRepo:       b.unitRepo,
		UnitManager:    b.unitManager,
		Validator:      b.validator,
	}
}
