package systemd

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/trly/quad-ops/internal/log"
)

// DBusConnection implements Connection interface wrapping systemd D-Bus operations.
type DBusConnection struct {
	conn *dbus.Conn
}

// NewDBusConnection creates a new D-Bus connection wrapper.
func NewDBusConnection(conn *dbus.Conn) *DBusConnection {
	return &DBusConnection{conn: conn}
}

// GetUnitProperty gets a property of a systemd unit.
func (d *DBusConnection) GetUnitProperty(ctx context.Context, unitName, propertyName string) (*dbus.Property, error) {
	prop, err := d.conn.GetUnitPropertyContext(ctx, unitName, propertyName)
	if err != nil {
		return nil, fmt.Errorf("error getting unit property %s for %s: %w", propertyName, unitName, err)
	}
	return prop, nil
}

// GetUnitProperties gets all properties of a systemd unit.
func (d *DBusConnection) GetUnitProperties(ctx context.Context, unitName string) (map[string]interface{}, error) {
	props, err := d.conn.GetUnitPropertiesContext(ctx, unitName)
	if err != nil {
		return nil, fmt.Errorf("error getting unit properties for %s: %w", unitName, err)
	}
	return props, nil
}

// StartUnit starts a systemd unit.
func (d *DBusConnection) StartUnit(ctx context.Context, unitName, mode string) (chan string, error) {
	ch := make(chan string)
	_, err := d.conn.StartUnitContext(ctx, unitName, mode, ch)
	if err != nil {
		return nil, fmt.Errorf("error starting unit %s: %w", unitName, err)
	}
	return ch, nil
}

// StopUnit stops a systemd unit.
func (d *DBusConnection) StopUnit(ctx context.Context, unitName, mode string) (chan string, error) {
	ch := make(chan string)
	_, err := d.conn.StopUnitContext(ctx, unitName, mode, ch)
	if err != nil {
		return nil, fmt.Errorf("error stopping unit %s: %w", unitName, err)
	}
	return ch, nil
}

// RestartUnit restarts a systemd unit.
func (d *DBusConnection) RestartUnit(ctx context.Context, unitName, mode string) (chan string, error) {
	ch := make(chan string)
	_, err := d.conn.RestartUnitContext(ctx, unitName, mode, ch)
	if err != nil {
		return nil, fmt.Errorf("error restarting unit %s: %w", unitName, err)
	}
	return ch, nil
}

// ResetFailedUnit resets the failed state of a unit.
func (d *DBusConnection) ResetFailedUnit(ctx context.Context, unitName string) error {
	err := d.conn.ResetFailedUnitContext(ctx, unitName)
	if err != nil {
		return fmt.Errorf("error resetting failed unit %s: %w", unitName, err)
	}
	return nil
}

// Reload reloads systemd configuration.
func (d *DBusConnection) Reload(ctx context.Context) error {
	err := d.conn.ReloadContext(ctx)
	if err != nil {
		return fmt.Errorf("error reloading systemd: %w", err)
	}
	return nil
}

// Close closes the D-Bus connection.
func (d *DBusConnection) Close() error {
	d.conn.Close()
	return nil
}

// DefaultConnectionFactory implements ConnectionFactory interface.
type DefaultConnectionFactory struct {
	logger log.Logger
}

// NewConnectionFactory creates a new connection factory with injected logger.
func NewConnectionFactory(logger log.Logger) *DefaultConnectionFactory {
	return &DefaultConnectionFactory{
		logger: logger,
	}
}

// NewConnection creates a new systemd connection based on configuration.
func (f *DefaultConnectionFactory) NewConnection(ctx context.Context, userMode bool) (Connection, error) {
	var conn *dbus.Conn
	var err error

	if userMode {
		f.logger.Debug("Establishing user connection to systemd")
		conn, err = dbus.NewUserConnectionContext(ctx)
	} else {
		f.logger.Debug("Establishing system connection to systemd")
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}

	if err != nil {
		return nil, NewConnectionError(userMode, err)
	}

	return NewDBusConnection(conn), nil
}
