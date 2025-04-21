package unit

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/trly/quad-ops/internal/config"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/ini.v1"

	"github.com/coreos/go-systemd/v22/dbus"
)

var ctx = context.Background()
var caser = cases.Title(language.English)

// SystemdUnit defines the interface for managing systemd units.
type SystemdUnit interface {
	// GetServiceName returns the full systemd service name
	GetServiceName() string

	// GetUnitType returns the type of the unit (container, volume, network, etc.)
	GetUnitType() string

	// GetUnitName returns the name of the unit
	GetUnitName() string

	// GetStatus returns the current status of the unit
	GetStatus() (string, error)

	// Start starts the unit
	Start() error

	// Stop stops the unit
	Stop() error

	// Restart restarts the unit
	Restart() error

	// Show displays the unit configuration and status
	Show() error
}

// BaseSystemdUnit provides common implementation for all systemd units.
type BaseSystemdUnit struct {
	Name string
	Type string
}

// GetServiceName returns the full systemd service name based on unit type.
func (u *BaseSystemdUnit) GetServiceName() string {
	if u.Type == "container" {
		return u.Name + ".service"
	}
	return u.Name + "-" + u.Type + ".service"
}

// GetUnitType returns the type of the unit.
func (u *BaseSystemdUnit) GetUnitType() string {
	return u.Type
}

// GetUnitName returns the name of the unit.
func (u *BaseSystemdUnit) GetUnitName() string {
	return u.Name
}

// GetStatus returns the current status of the unit.
func (u *BaseSystemdUnit) GetStatus() (string, error) {
	conn, err := getSystemdConnection()
	if err != nil {
		return "", fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	prop, err := conn.GetUnitPropertyContext(ctx, serviceName, "ActiveState")
	if err != nil {
		return "", fmt.Errorf("error getting unit property: %w", err)
	}
	return prop.Value.Value().(string), nil
}

// Start starts the unit.
func (u *BaseSystemdUnit) Start() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	ch := make(chan string)
	_, err = conn.StartUnitContext(context.Background(), serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("error starting unit %s: %w", serviceName, err)
	}

	if config.GetConfig().Verbose {
		log.Printf("successfully started unit %s\n", serviceName)
	}
	return nil
}

// Stop stops the unit.
func (u *BaseSystemdUnit) Stop() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	ch := make(chan string)
	_, err = conn.StopUnitContext(context.Background(), serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("error stopping unit %s: %w", serviceName, err)
	}

	if config.GetConfig().Verbose {
		log.Printf("successfully stopped unit %s\n", serviceName)
	}
	return nil
}

// Restart restarts the unit.
func (u *BaseSystemdUnit) Restart() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("error restarting unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		return fmt.Errorf("unit restart failed: %s", result)
	}

	if config.GetConfig().Verbose {
		log.Printf("successfully restarted unit %s\n", serviceName)
	}
	return nil
}

// Show displays the unit configuration and status.
func (u *BaseSystemdUnit) Show() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	prop, err := conn.GetUnitPropertiesContext(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("error getting unit properties: %w", err)
	}

	fmt.Printf("\n=== %s ===\n\n", serviceName)

	fmt.Println("Status:")
	fmt.Printf("  %-20s: %v\n", "State", prop["ActiveState"])
	fmt.Printf("  %-20s: %v\n", "Sub-State", prop["SubState"])
	fmt.Printf("  %-20s: %v\n", "Load State", prop["LoadState"])

	fmt.Println("\nUnit Information:")
	fmt.Printf("  %-20s: %v\n", "Description", prop["Description"])
	fmt.Printf("  %-20s: %v\n", "Path", prop["FragmentPath"])

	// Read and display the actual quadlet configuration
	if fragmentPath, ok := prop["FragmentPath"].(string); ok {
		content, err := os.ReadFile(fragmentPath) //nolint:gosec // Safe as path comes from systemd D-Bus interface, not user input
		if err == nil {
			unitConfig, _ := ini.Load(content)
			sectionName := fmt.Sprintf("X-%s", caser.String(u.Type))
			if section, err := unitConfig.GetSection(sectionName); err == nil {
				fmt.Printf("\n%s Configuration:\n", caser.String(u.Type))
				for _, key := range section.Keys() {
					fmt.Printf("  %-20s: %s\n", key.Name(), key.Value())
				}
			}
		}
	}

	fmt.Println()
	return nil
}

// ReloadSystemd reloads the systemd configuration.
func ReloadSystemd() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	if config.GetConfig().Verbose {
		log.Printf("reloading systemd...")
	}
	err = conn.ReloadContext(ctx)
	if err != nil {
		return fmt.Errorf("error reloading systemd: %w", err)
	}

	return nil
}

// GetUnitStatus - Legacy function for backward compatibility.
func GetUnitStatus(unitName string, unitType string) (string, error) {
	unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
	return unit.GetStatus()
}

// RestartUnit - Legacy function for backward compatibility.
func RestartUnit(unitName string, unitType string) error {
	unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
	return unit.Restart()
}

// StartUnit - Legacy function for backward compatibility.
func StartUnit(unitName string, unitType string) error {
	unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
	return unit.Start()
}

// StopUnit - Legacy function for backward compatibility.
func StopUnit(unitName string, unitType string) error {
	unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
	return unit.Stop()
}

// ShowUnit - Legacy function for backward compatibility.
func ShowUnit(unitName string, unitType string) error {
	unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
	return unit.Show()
}

func getSystemdConnection() (*dbus.Conn, error) {
	if config.GetConfig().Verbose {
		if config.GetConfig().UserMode {
			log.Printf("establishing user connection to systemd...")
		} else {
			log.Printf("establishing system connection to systemd...")
		}
	}

	if config.GetConfig().UserMode {
		return dbus.NewUserConnectionContext(ctx)
	}
	return dbus.NewSystemConnectionContext(ctx)
}
