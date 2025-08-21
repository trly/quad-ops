package systemd

import (
	"context"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/dependency"
)

func TestDefaultUnitManager(t *testing.T) {
	t.Run("GetStatus returns unit status", func(t *testing.T) {
		// Setup mock connection that returns "active"
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, _ string) (*dbus.Property, error) {
				return createMockProperty("active"), nil
			},
		}

		// Setup unit manager with mocks
		unitManager := createTestUnitManager(mockConn)

		// Test GetStatus
		status, err := unitManager.GetStatus("test-unit", "container")
		require.NoError(t, err)
		assert.Equal(t, "active", status)
	})

	t.Run("Start calls connection StartUnit", func(t *testing.T) {
		startCalled := false
		mockConn := &MockConnection{
			StartUnitFunc: func(_ context.Context, unitName, mode string) (chan string, error) {
				startCalled = true
				assert.Equal(t, "test-unit.service", unitName)
				assert.Equal(t, "replace", mode)
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}

		unitManager := createTestUnitManager(mockConn)

		err := unitManager.Start("test-unit", "container")
		require.NoError(t, err)
		assert.True(t, startCalled)
	})

	t.Run("Stop calls connection StopUnit", func(t *testing.T) {
		stopCalled := false
		mockConn := &MockConnection{
			StopUnitFunc: func(_ context.Context, unitName, mode string) (chan string, error) {
				stopCalled = true
				assert.Equal(t, "test-unit.service", unitName)
				assert.Equal(t, "replace", mode)
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}

		unitManager := createTestUnitManager(mockConn)

		err := unitManager.Stop("test-unit", "container")
		require.NoError(t, err)
		assert.True(t, stopCalled)
	})

	t.Run("ReloadSystemd calls connection Reload", func(t *testing.T) {
		reloadCalled := false
		mockConn := &MockConnection{
			ReloadFunc: func(_ context.Context) error {
				reloadCalled = true
				return nil
			},
		}

		unitManager := createTestUnitManager(mockConn)

		err := unitManager.ReloadSystemd()
		require.NoError(t, err)
		assert.True(t, reloadCalled)
	})

	t.Run("GetUnit returns ManagedUnit with dependencies", func(t *testing.T) {
		mockConn := &MockConnection{}
		unitManager := createTestUnitManager(mockConn)

		unit := unitManager.GetUnit("test-unit", "container")

		// Verify unit is a ManagedUnit
		managedUnit, ok := unit.(*ManagedUnit)
		require.True(t, ok)

		// Verify properties
		assert.Equal(t, "test-unit", managedUnit.GetUnitName())
		assert.Equal(t, "container", managedUnit.GetUnitType())
		assert.Equal(t, "test-unit.service", managedUnit.GetServiceName())
	})
}

func TestDefaultOrchestrator(t *testing.T) {
	t.Run("StartUnitDependencyAware handles different unit types", func(t *testing.T) {
		mockUnitManager := &MockUnitManager{
			StartFunc: func(unitName, unitType string) error {
				assert.Equal(t, "test-unit", unitName)
				assert.Equal(t, "network", unitType)
				return nil
			},
		}

		orchestrator := NewDefaultOrchestrator(mockUnitManager)
		dependencyGraph := dependency.NewServiceDependencyGraph()

		// Test one-shot service (network)
		err := orchestrator.StartUnitDependencyAware("test-unit", "network", dependencyGraph)
		require.NoError(t, err)
	})

	t.Run("StartUnitDependencyAware handles container with dependencies", func(t *testing.T) {
		restartCalled := false
		mockUnitManager := &MockUnitManager{
			RestartFunc: func(unitName, unitType string) error {
				restartCalled = true
				assert.Equal(t, "project-service", unitName)
				assert.Equal(t, "container", unitType)
				return nil
			},
		}

		orchestrator := NewDefaultOrchestrator(mockUnitManager)

		// Create dependency graph with service
		dependencyGraph := dependency.NewServiceDependencyGraph()
		_ = dependencyGraph.AddService("service")

		err := orchestrator.StartUnitDependencyAware("project-service", "container", dependencyGraph)
		require.NoError(t, err)
		assert.True(t, restartCalled)
	})

	t.Run("RestartChangedUnits reloads systemd first", func(t *testing.T) {
		reloadCalled := false
		mockUnitManager := &MockUnitManager{
			ReloadSystemdFunc: func() error {
				reloadCalled = true
				return nil
			},
		}

		orchestrator := NewDefaultOrchestrator(mockUnitManager)

		// Create some unit changes
		changes := []UnitChange{
			{
				Name: "test-network",
				Type: "network",
				Unit: &MockUnit{},
			},
		}

		err := orchestrator.RestartChangedUnits(changes, nil)
		require.NoError(t, err)
		assert.True(t, reloadCalled)
	})
}

func TestDefaultContextProvider(t *testing.T) {
	t.Run("GetContext returns valid context", func(t *testing.T) {
		provider := NewDefaultContextProvider()
		ctx := provider.GetContext()

		assert.NotNil(t, ctx)
		assert.Equal(t, context.Background(), ctx)
	})
}

func TestDefaultTextCaser(t *testing.T) {
	t.Run("Title converts text to title case", func(t *testing.T) {
		caser := NewDefaultTextCaser()

		tests := []struct {
			input    string
			expected string
		}{
			{"container", "Container"},
			{"network", "Network"},
			{"volume", "Volume"},
			{"build", "Build"},
			{"", ""},
		}

		for _, test := range tests {
			result := caser.Title(test.input)
			assert.Equal(t, test.expected, result)
		}
	})
}

func TestDefaultFactory(t *testing.T) {
	t.Run("NewDefaultFactory creates all components", func(t *testing.T) {
		factory := NewDefaultFactory()

		assert.NotNil(t, factory.GetConnectionFactory())
		assert.NotNil(t, factory.GetContextProvider())
		assert.NotNil(t, factory.GetTextCaser())
		assert.NotNil(t, factory.GetUnitManager())
		assert.NotNil(t, factory.GetOrchestrator())
	})

	t.Run("GetDefaultUnitManager returns unit manager", func(t *testing.T) {
		unitManager := GetDefaultUnitManager()
		assert.NotNil(t, unitManager)
	})

	t.Run("GetDefaultOrchestrator returns orchestrator", func(t *testing.T) {
		orchestrator := GetDefaultOrchestrator()
		assert.NotNil(t, orchestrator)
	})
}

// Helper function to create a test unit manager with mocked connection.
func createTestUnitManager(mockConn Connection) UnitManager {
	mockFactory := &MockConnectionFactory{
		Connection: mockConn,
	}
	contextProvider := NewDefaultContextProvider()
	textCaser := NewDefaultTextCaser()

	return NewDefaultUnitManager(mockFactory, contextProvider, textCaser)
}
