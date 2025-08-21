package systemd

import (
	"context"
	"errors"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

func TestManagedUnit(t *testing.T) {
	// Initialize config and logger for tests
	configProvider := config.NewDefaultConfigProvider()
	configProvider.InitConfig()

	t.Run("NewManagedUnit creates unit with dependencies", func(t *testing.T) {
		mockFactory := &MockConnectionFactory{}
		contextProvider := NewDefaultContextProvider()
		textCaser := NewDefaultTextCaser()
		configProvider := config.NewConfigProvider()
		logger := log.NewLogger(false)

		unit := NewManagedUnit("test-unit", "container", mockFactory, contextProvider, textCaser, configProvider, logger)

		assert.Equal(t, "test-unit", unit.GetUnitName())
		assert.Equal(t, "container", unit.GetUnitType())
		assert.Equal(t, "test-unit.service", unit.GetServiceName())
	})

	t.Run("GetStatus returns status from connection", func(t *testing.T) {
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, _ string) (*dbus.Property, error) {
				return &dbus.Property{Value: godbus.MakeVariant("active")}, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		status, err := unit.GetStatus()
		require.NoError(t, err)
		assert.Equal(t, "active", status)
	})

	t.Run("GetStatus returns error when connection fails", func(t *testing.T) {
		mockFactory := &MockConnectionFactory{
			NewConnectionFunc: func(_ context.Context, _ bool) (Connection, error) {
				return nil, NewConnectionError(false, errors.New("connection failed"))
			},
		}

		unit := createTestManagedUnit(mockFactory)

		status, err := unit.GetStatus()
		assert.Error(t, err)
		assert.Empty(t, status)
		assert.True(t, IsConnectionError(err))
	})

	t.Run("GetStatus returns Error when property retrieval fails", func(t *testing.T) {
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, _ string) (*dbus.Property, error) {
				return nil, errors.New("unit not found")
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		status, err := unit.GetStatus()
		assert.Error(t, err)
		assert.Empty(t, status)
		assert.True(t, IsError(err))

		systemdErr := err.(*Error)
		assert.Equal(t, "GetStatus", systemdErr.Operation)
		assert.Equal(t, "test-unit", systemdErr.UnitName)
		assert.Equal(t, "container", systemdErr.UnitType)
	})

	t.Run("Start successfully starts unit", func(t *testing.T) {
		mockConn := &MockConnection{
			StartUnitFunc: func(_ context.Context, unitName, mode string) (chan string, error) {
				assert.Equal(t, "test-unit.service", unitName)
				assert.Equal(t, "replace", mode)
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		err := unit.Start()
		require.NoError(t, err)
	})

	t.Run("Start handles non-done result gracefully", func(t *testing.T) {
		mockConn := &MockConnection{
			StartUnitFunc: func(_ context.Context, _, _ string) (chan string, error) {
				ch := make(chan string, 1)
				ch <- "failed" // Simulate failed result
				close(ch)
				return ch, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		err := unit.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unit start failed")
	})

	t.Run("Stop successfully stops unit", func(t *testing.T) {
		mockConn := &MockConnection{
			StopUnitFunc: func(_ context.Context, unitName, mode string) (chan string, error) {
				assert.Equal(t, "test-unit.service", unitName)
				assert.Equal(t, "replace", mode)
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		err := unit.Stop()
		require.NoError(t, err)
	})

	t.Run("Restart checks load state before restarting", func(t *testing.T) {
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, propertyName string) (*dbus.Property, error) {
				if propertyName == "LoadState" {
					return &dbus.Property{Value: godbus.MakeVariant("loaded")}, nil
				}
				return &dbus.Property{Value: godbus.MakeVariant("unknown")}, nil
			},
			RestartUnitFunc: func(_ context.Context, _, _ string) (chan string, error) {
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		err := unit.Restart()
		require.NoError(t, err)
	})

	t.Run("Restart fails when unit not loaded", func(t *testing.T) {
		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, _, propertyName string) (*dbus.Property, error) {
				if propertyName == "LoadState" {
					return &dbus.Property{Value: godbus.MakeVariant("not-found")}, nil
				}
				return &dbus.Property{Value: godbus.MakeVariant("unknown")}, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		err := unit.Restart()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not loaded")
	})

	t.Run("ResetFailed successfully resets unit", func(t *testing.T) {
		mockConn := &MockConnection{
			ResetFailedUnitFunc: func(_ context.Context, unitName string) error {
				assert.Equal(t, "test-unit.service", unitName)
				return nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		err := unit.ResetFailed()
		require.NoError(t, err)
	})

	t.Run("Show displays unit information", func(t *testing.T) {
		mockConn := &MockConnection{
			GetUnitPropertiesFunc: func(_ context.Context, unitName string) (map[string]interface{}, error) {
				assert.Equal(t, "test-unit.service", unitName)
				return map[string]interface{}{
					"ActiveState":  "active",
					"SubState":     "running",
					"LoadState":    "loaded",
					"Description":  "Test Unit",
					"FragmentPath": "/etc/systemd/system/test-unit.service",
				}, nil
			},
		}
		mockFactory := &MockConnectionFactory{Connection: mockConn}

		unit := createTestManagedUnit(mockFactory)

		// Show() prints to stdout, so we can't easily test the output
		// But we can verify it doesn't error
		err := unit.Show()
		require.NoError(t, err)
	})
}

// Helper function to create a test managed unit.
func createTestManagedUnit(factory ConnectionFactory) *ManagedUnit {
	contextProvider := NewDefaultContextProvider()
	textCaser := NewDefaultTextCaser()
	configProvider := config.NewConfigProvider()
	logger := log.NewLogger(false)
	return NewManagedUnit("test-unit", "container", factory, contextProvider, textCaser, configProvider, logger)
}
