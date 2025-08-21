package systemd

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/trly/quad-ops/internal/dependency"
)

// MockConnection implements Connection interface for testing.
type MockConnection struct {
	GetUnitPropertyFunc   func(ctx context.Context, unitName, propertyName string) (*dbus.Property, error)
	GetUnitPropertiesFunc func(ctx context.Context, unitName string) (map[string]interface{}, error)
	StartUnitFunc         func(ctx context.Context, unitName, mode string) (chan string, error)
	StopUnitFunc          func(ctx context.Context, unitName, mode string) (chan string, error)
	RestartUnitFunc       func(ctx context.Context, unitName, mode string) (chan string, error)
	ResetFailedUnitFunc   func(ctx context.Context, unitName string) error
	ReloadFunc            func(ctx context.Context) error
	CloseFunc             func() error
}

// GetUnitProperty gets a property of a systemd unit.
func (m *MockConnection) GetUnitProperty(ctx context.Context, unitName, propertyName string) (*dbus.Property, error) {
	if m.GetUnitPropertyFunc != nil {
		return m.GetUnitPropertyFunc(ctx, unitName, propertyName)
	}
	return nil, fmt.Errorf("mock not implemented")
}

// GetUnitProperties gets all properties of a systemd unit.
func (m *MockConnection) GetUnitProperties(ctx context.Context, unitName string) (map[string]interface{}, error) {
	if m.GetUnitPropertiesFunc != nil {
		return m.GetUnitPropertiesFunc(ctx, unitName)
	}
	return nil, fmt.Errorf("mock not implemented")
}

// StartUnit starts a systemd unit.
func (m *MockConnection) StartUnit(ctx context.Context, unitName, mode string) (chan string, error) {
	if m.StartUnitFunc != nil {
		return m.StartUnitFunc(ctx, unitName, mode)
	}
	return nil, fmt.Errorf("mock not implemented")
}

// StopUnit stops a systemd unit.
func (m *MockConnection) StopUnit(ctx context.Context, unitName, mode string) (chan string, error) {
	if m.StopUnitFunc != nil {
		return m.StopUnitFunc(ctx, unitName, mode)
	}
	return nil, fmt.Errorf("mock not implemented")
}

// RestartUnit restarts a systemd unit.
func (m *MockConnection) RestartUnit(ctx context.Context, unitName, mode string) (chan string, error) {
	if m.RestartUnitFunc != nil {
		return m.RestartUnitFunc(ctx, unitName, mode)
	}
	return nil, fmt.Errorf("mock not implemented")
}

// ResetFailedUnit resets the failed state of a unit.
func (m *MockConnection) ResetFailedUnit(ctx context.Context, unitName string) error {
	if m.ResetFailedUnitFunc != nil {
		return m.ResetFailedUnitFunc(ctx, unitName)
	}
	return fmt.Errorf("mock not implemented")
}

// Reload reloads systemd configuration.
func (m *MockConnection) Reload(ctx context.Context) error {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(ctx)
	}
	return fmt.Errorf("mock not implemented")
}

// Close closes the connection.
func (m *MockConnection) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockConnectionFactory implements ConnectionFactory interface for testing.
type MockConnectionFactory struct {
	NewConnectionFunc func(ctx context.Context, userMode bool) (Connection, error)
	Connection        Connection
}

// NewConnection creates a new systemd connection based on configuration.
func (m *MockConnectionFactory) NewConnection(ctx context.Context, userMode bool) (Connection, error) {
	if m.NewConnectionFunc != nil {
		return m.NewConnectionFunc(ctx, userMode)
	}
	if m.Connection != nil {
		return m.Connection, nil
	}
	return nil, fmt.Errorf("mock not configured")
}

// MockContextProvider implements ContextProvider interface for testing.
type MockContextProvider struct {
	GetContextFunc func() context.Context
	Ctx            context.Context
}

// GetContext returns a context for systemd operations.
func (m *MockContextProvider) GetContext() context.Context {
	if m.GetContextFunc != nil {
		return m.GetContextFunc()
	}
	if m.Ctx != nil {
		return m.Ctx
	}
	return context.Background()
}

// MockTextCaser implements TextCaser interface for testing.
type MockTextCaser struct {
	TitleFunc func(text string) string
}

// Title converts text to title case.
func (m *MockTextCaser) Title(text string) string {
	if m.TitleFunc != nil {
		return m.TitleFunc(text)
	}
	// Default simple title case implementation for testing
	if len(text) == 0 {
		return text
	}
	return string(text[0]-32) + text[1:] // Simple uppercase first character
}

// MockUnitManager implements UnitManager interface for testing.
type MockUnitManager struct {
	GetUnitFunc               func(name, unitType string) Unit
	GetStatusFunc             func(unitName, unitType string) (string, error)
	StartFunc                 func(unitName, unitType string) error
	StopFunc                  func(unitName, unitType string) error
	RestartFunc               func(unitName, unitType string) error
	ShowFunc                  func(unitName, unitType string) error
	ResetFailedFunc           func(unitName, unitType string) error
	ReloadSystemdFunc         func() error
	GetUnitFailureDetailsFunc func(unitName string) string
}

// GetUnit creates a Unit interface for the given name and type.
func (m *MockUnitManager) GetUnit(name, unitType string) Unit {
	if m.GetUnitFunc != nil {
		return m.GetUnitFunc(name, unitType)
	}
	return NewBaseUnit(name, unitType)
}

// GetStatus returns the current status of a unit.
func (m *MockUnitManager) GetStatus(unitName, unitType string) (string, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(unitName, unitType)
	}
	return "inactive", nil
}

// Start starts a unit.
func (m *MockUnitManager) Start(unitName, unitType string) error {
	if m.StartFunc != nil {
		return m.StartFunc(unitName, unitType)
	}
	return nil
}

// Stop stops a unit.
func (m *MockUnitManager) Stop(unitName, unitType string) error {
	if m.StopFunc != nil {
		return m.StopFunc(unitName, unitType)
	}
	return nil
}

// Restart restarts a unit.
func (m *MockUnitManager) Restart(unitName, unitType string) error {
	if m.RestartFunc != nil {
		return m.RestartFunc(unitName, unitType)
	}
	return nil
}

// Show displays unit configuration and status.
func (m *MockUnitManager) Show(unitName, unitType string) error {
	if m.ShowFunc != nil {
		return m.ShowFunc(unitName, unitType)
	}
	return nil
}

// ResetFailed resets the failed state of a unit.
func (m *MockUnitManager) ResetFailed(unitName, unitType string) error {
	if m.ResetFailedFunc != nil {
		return m.ResetFailedFunc(unitName, unitType)
	}
	return nil
}

// ReloadSystemd reloads systemd configuration.
func (m *MockUnitManager) ReloadSystemd() error {
	if m.ReloadSystemdFunc != nil {
		return m.ReloadSystemdFunc()
	}
	return nil
}

// GetUnitFailureDetails gets detailed failure information for a unit.
func (m *MockUnitManager) GetUnitFailureDetails(unitName string) string {
	if m.GetUnitFailureDetailsFunc != nil {
		return m.GetUnitFailureDetailsFunc(unitName)
	}
	return "Unit: " + unitName + "\n  Status: Mock failure details"
}

// MockOrchestrator implements Orchestrator interface for testing.
type MockOrchestrator struct {
	StartUnitDependencyAwareFunc func(unitName, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error
	RestartChangedUnitsFunc      func(changedUnits []UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error
}

// StartUnitDependencyAware starts or restarts a unit with dependency awareness.
func (m *MockOrchestrator) StartUnitDependencyAware(unitName, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error {
	if m.StartUnitDependencyAwareFunc != nil {
		return m.StartUnitDependencyAwareFunc(unitName, unitType, dependencyGraph)
	}
	return nil
}

// RestartChangedUnits restarts all changed units in dependency-aware order.
func (m *MockOrchestrator) RestartChangedUnits(changedUnits []UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	if m.RestartChangedUnitsFunc != nil {
		return m.RestartChangedUnitsFunc(changedUnits, projectDependencyGraphs)
	}
	return nil
}

// MockUnit implements Unit interface for testing.
type MockUnit struct {
	GetServiceNameFunc string
	GetUnitTypeFunc    string
	GetUnitNameFunc    string
	GetStatusFunc      func() (string, error)
	StartFunc          func() error
	StopFunc           func() error
	RestartFunc        func() error
	ShowFunc           func() error
	ResetFailedFunc    func() error
}

// GetServiceName returns the full systemd service name.
func (m *MockUnit) GetServiceName() string {
	if m.GetServiceNameFunc != "" {
		return m.GetServiceNameFunc
	}
	return "mock.service"
}

// GetUnitType returns the type of the unit.
func (m *MockUnit) GetUnitType() string {
	if m.GetUnitTypeFunc != "" {
		return m.GetUnitTypeFunc
	}
	return "container"
}

// GetUnitName returns the name of the unit.
func (m *MockUnit) GetUnitName() string {
	if m.GetUnitNameFunc != "" {
		return m.GetUnitNameFunc
	}
	return "mock"
}

// GetStatus returns the current status of the unit.
func (m *MockUnit) GetStatus() (string, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc()
	}
	return "inactive", nil
}

// Start starts the unit.
func (m *MockUnit) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

// Stop stops the unit.
func (m *MockUnit) Stop() error {
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

// Restart restarts the unit.
func (m *MockUnit) Restart() error {
	if m.RestartFunc != nil {
		return m.RestartFunc()
	}
	return nil
}

// Show displays the unit configuration and status.
func (m *MockUnit) Show() error {
	if m.ShowFunc != nil {
		return m.ShowFunc()
	}
	return nil
}

// ResetFailed resets the failed state of the unit.
func (m *MockUnit) ResetFailed() error {
	if m.ResetFailedFunc != nil {
		return m.ResetFailedFunc()
	}
	return nil
}
