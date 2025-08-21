package systemd

import (
	"context"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/testutil"
	"github.com/trly/quad-ops/internal/testutil/fakerunner"
)

// TestDependencyInjectionIntegration verifies that dependency injection is working correctly
// throughout the systemd package and that interfaces are properly implemented.
func TestDependencyInjectionIntegration(t *testing.T) {
	t.Run("DefaultFactory creates properly injected components", func(t *testing.T) {
		configProvider := testutil.NewMockConfig(t)
		logger := testutil.NewTestLogger(t)
		factory := NewDefaultFactory(configProvider, logger)

		// Test that all components are created and injectable
		assert.NotNil(t, factory.GetConnectionFactory())
		assert.NotNil(t, factory.GetContextProvider())
		assert.NotNil(t, factory.GetTextCaser())
		assert.NotNil(t, factory.GetUnitManager())
		assert.NotNil(t, factory.GetOrchestrator())
	})

	t.Run("MockConnectionFactory works properly", func(t *testing.T) {
		mockConn := &MockConnection{}
		mockFactory := &MockConnectionFactory{
			Connection: mockConn,
		}

		conn, err := mockFactory.NewConnection(context.Background(), false)
		require.NoError(t, err)
		assert.Equal(t, mockConn, conn)
	})

	t.Run("UnitManager with mock dependencies works", func(t *testing.T) {
		// Setup mock connection that returns "active" status
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, _ string) (*dbus.Property, error) {
				// Create a mock property that returns "active"
				return createMockProperty("active"), nil
			},
		}

		// Setup mock factory
		mockFactory := &MockConnectionFactory{
			Connection: mockConn,
		}

		// Setup providers
		contextProvider := NewDefaultContextProvider()
		textCaser := NewDefaultTextCaser()

		// Create unit manager with mocked dependencies
		configProvider2 := testutil.NewMockConfig(t)
		logger := testutil.NewTestLogger(t)
		runner := fakerunner.New()
		unitManager := NewDefaultUnitManager(mockFactory, contextProvider, textCaser, configProvider2, logger, runner)

		// Test that the unit manager can get status using mocked connection
		status, err := unitManager.GetStatus("test-unit", "container")
		require.NoError(t, err)
		assert.Equal(t, "active", status)
	})

	t.Run("ManagedUnit with mock dependencies works", func(t *testing.T) {
		// Setup mock connection that returns "inactive" status
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, _ string) (*dbus.Property, error) {
				// Create a mock property that returns "inactive"
				return createMockProperty("inactive"), nil
			},
		}

		// Setup mock factory
		mockFactory := &MockConnectionFactory{
			Connection: mockConn,
		}

		// Setup providers
		contextProvider := NewDefaultContextProvider()
		textCaser := NewDefaultTextCaser()

		// Create managed unit with mocked dependencies
		configProvider3 := testutil.NewMockConfig(t)
		logger2 := testutil.NewTestLogger(t)
		unit := NewManagedUnit("test-unit", "container", mockFactory, contextProvider, textCaser, configProvider3, logger2)

		// Test that the unit can get status using mocked connection
		status, err := unit.GetStatus()
		require.NoError(t, err)
		assert.Equal(t, "inactive", status)
	})

	t.Run("TextCaser interface works", func(t *testing.T) {
		textCaser := NewDefaultTextCaser()
		result := textCaser.Title("container")
		assert.Equal(t, "Container", result)
	})

	t.Run("ContextProvider interface works", func(t *testing.T) {
		contextProvider := NewDefaultContextProvider()
		ctx := contextProvider.GetContext()
		assert.NotNil(t, ctx)
	})

	t.Run("Orchestrator with mock dependencies works", func(t *testing.T) {
		// Setup mock unit manager
		mockUnitManager := &MockUnitManager{
			ReloadSystemdFunc: func() error {
				return nil
			},
		}

		// Create orchestrator with mocked unit manager
		configProvider := testutil.NewMockConfig(t)
		logger := testutil.NewTestLogger(t)
		connectionFactory := NewConnectionFactory(logger)
		orchestrator := NewDefaultOrchestrator(mockUnitManager, connectionFactory, configProvider, logger)

		// Test basic operation
		assert.NotNil(t, orchestrator)
	})
}

// Helper function to create a mock dbus property.
func createMockProperty(value string) *dbus.Property {
	// Use the real dbus library to create a proper variant
	variant := godbus.MakeVariant(value)
	return &dbus.Property{Value: variant}
}
