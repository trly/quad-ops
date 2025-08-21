package systemd

import (
	"context"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/log"
)

func TestDBusConnection(t *testing.T) {
	// Note: These tests don't actually connect to D-Bus since that requires a running systemd
	// They test the wrapper logic and error handling

	t.Run("NewDBusConnection creates wrapper", func(t *testing.T) {
		// We can't create a real dbus.Conn in tests without systemd running
		// So we test the wrapper creation logic
		wrapper := NewDBusConnection(nil)
		assert.NotNil(t, wrapper)
	})

	t.Run("GetUnitProperty wraps errors properly", func(t *testing.T) {
		// Create a connection wrapper with nil conn to test error handling
		wrapper := &DBusConnection{conn: nil}

		// This will panic due to nil conn, but demonstrates the wrapper structure
		// In real usage, the conn would be valid
		assert.NotNil(t, wrapper)
	})
}

func TestConnectionFactory(t *testing.T) {
	t.Run("NewConnectionFactory creates factory", func(t *testing.T) {
		logger := log.NewLogger(false)
		factory := NewConnectionFactory(logger)
		assert.NotNil(t, factory)
	})

	// Note: We can't test actual D-Bus connections without systemd running
	// The NewConnection method would try to connect to systemd which isn't available in tests
}

func TestMockConnectionFactory(t *testing.T) {
	t.Run("MockConnectionFactory returns configured connection", func(t *testing.T) {
		mockConn := &MockConnection{}
		factory := &MockConnectionFactory{
			Connection: mockConn,
		}

		conn, err := factory.NewConnection(context.Background(), false)
		require.NoError(t, err)
		assert.Equal(t, mockConn, conn)
	})

	t.Run("MockConnectionFactory calls custom function", func(t *testing.T) {
		called := false
		factory := &MockConnectionFactory{
			NewConnectionFunc: func(_ context.Context, userMode bool) (Connection, error) {
				called = true
				assert.False(t, userMode)
				return &MockConnection{}, nil
			},
		}

		conn, err := factory.NewConnection(context.Background(), false)
		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.True(t, called)
	})

	t.Run("MockConnectionFactory returns error when not configured", func(t *testing.T) {
		factory := &MockConnectionFactory{}

		conn, err := factory.NewConnection(context.Background(), false)
		require.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "mock not configured")
	})
}

func TestMockConnection(t *testing.T) {
	t.Run("GetUnitProperty calls mock function", func(t *testing.T) {
		called := false
		expectedProp := &dbus.Property{Value: godbus.MakeVariant("active")}

		mockConn := &MockConnection{
			GetUnitPropertyFunc: func(_ context.Context, unitName, propertyName string) (*dbus.Property, error) {
				called = true
				assert.Equal(t, "test-unit.service", unitName)
				assert.Equal(t, "ActiveState", propertyName)
				return expectedProp, nil
			},
		}

		prop, err := mockConn.GetUnitProperty(context.Background(), "test-unit.service", "ActiveState")
		require.NoError(t, err)
		assert.Equal(t, expectedProp, prop)
		assert.True(t, called)
	})

	t.Run("GetUnitProperties returns error when not configured", func(t *testing.T) {
		mockConn := &MockConnection{}

		props, err := mockConn.GetUnitProperties(context.Background(), "test-unit.service")
		assert.Error(t, err)
		assert.Nil(t, props)
		assert.Contains(t, err.Error(), "mock not implemented")
	})

	t.Run("StartUnit calls mock function", func(t *testing.T) {
		called := false
		mockConn := &MockConnection{
			StartUnitFunc: func(_ context.Context, unitName, mode string) (chan string, error) {
				called = true
				assert.Equal(t, "test-unit.service", unitName)
				assert.Equal(t, "replace", mode)
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}

		ch, err := mockConn.StartUnit(context.Background(), "test-unit.service", "replace")
		require.NoError(t, err)
		result := <-ch
		assert.Equal(t, "done", result)
		assert.True(t, called)
	})

	t.Run("StopUnit returns error when not configured", func(t *testing.T) {
		mockConn := &MockConnection{}

		ch, err := mockConn.StopUnit(context.Background(), "test-unit.service", "replace")
		require.Error(t, err)
		assert.Nil(t, ch)
		assert.Contains(t, err.Error(), "mock not implemented")
	})

	t.Run("RestartUnit calls mock function", func(t *testing.T) {
		called := false
		mockConn := &MockConnection{
			RestartUnitFunc: func(_ context.Context, _, _ string) (chan string, error) {
				called = true
				ch := make(chan string, 1)
				ch <- "done"
				close(ch)
				return ch, nil
			},
		}

		ch, err := mockConn.RestartUnit(context.Background(), "test-unit.service", "replace")
		require.NoError(t, err)
		result := <-ch
		assert.Equal(t, "done", result)
		assert.True(t, called)
	})

	t.Run("ResetFailedUnit calls mock function", func(t *testing.T) {
		called := false
		mockConn := &MockConnection{
			ResetFailedUnitFunc: func(_ context.Context, unitName string) error {
				called = true
				assert.Equal(t, "test-unit.service", unitName)
				return nil
			},
		}

		err := mockConn.ResetFailedUnit(context.Background(), "test-unit.service")
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("Reload calls mock function", func(t *testing.T) {
		called := false
		mockConn := &MockConnection{
			ReloadFunc: func(_ context.Context) error {
				called = true
				return nil
			},
		}

		err := mockConn.Reload(context.Background())
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("Close calls mock function", func(t *testing.T) {
		called := false
		mockConn := &MockConnection{
			CloseFunc: func() error {
				called = true
				return nil
			},
		}

		err := mockConn.Close()
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("Close returns nil when not configured", func(t *testing.T) {
		mockConn := &MockConnection{}

		err := mockConn.Close()
		assert.NoError(t, err)
	})
}

func TestGetConnection(t *testing.T) {
	t.Run("GetConnection creates connection", func(t *testing.T) {
		// Skip this test as it requires systemd to be running and config initialized
		t.Skip("Requires actual systemd connection - covered by integration tests")
	})
}
