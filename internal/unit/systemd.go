package unit

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/trly/quad-ops/internal/config"
	"gopkg.in/ini.v1"

	"github.com/coreos/go-systemd/v22/dbus"
)

var ctx = context.Background()

func GetUnitStatus(unitName string, unitType string) (string, error) {
	conn, err := getSystemdConnection()
	if err != nil {
		return "", fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	quadletService := unitName
	if unitType == "container" {
		quadletService += ".service"
	} else {
		quadletService += "-" + unitType + ".service"
	}

	prop, err := conn.GetUnitPropertyContext(ctx, quadletService, "ActiveState")
	if err != nil {
		return "", fmt.Errorf("error getting unit property: %w", err)
	}
	return prop.Value.Value().(string), nil
}

func RestartUnit(unitName string, unitType string) error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	quadletService := unitName
	if unitType == "container" {
		quadletService += ".service"
	} else {
		quadletService += "-" + unitType + ".service"
	}

	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), quadletService, "replace", ch)
	if err != nil {
		return fmt.Errorf("error restarting unit %s: %w", quadletService, err)
	}

	result := <-ch
	if result != "done" {
		return fmt.Errorf("unit restart failed: %s", result)
	}
	if config.GetConfig().Verbose {
		log.Printf("successfully restarted unit %s\n", quadletService)
	}

	return nil
}

func StartUnit(unitName string, unitType string) error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()
	quadletService := unitName
	if unitType == "container" {
		quadletService += ".service"
	} else {
		quadletService += "-" + unitType + ".service"
	}
	ch := make(chan string)
	_, err = conn.StartUnitContext(context.Background(), quadletService, "replace", ch)
	if err != nil {
		return fmt.Errorf("error starting unit %s: %w", quadletService, err)
	}
	if config.GetConfig().Verbose {
		log.Printf("successfully started unit %s\n", quadletService)
	}
	return nil
}

func StopUnit(unitName string, unitType string) error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()
	quadletService := unitName
	if unitType == "container" {
		quadletService += ".service"
	} else {
		quadletService += "-" + unitType + ".service"
	}
	ch := make(chan string)
	_, err = conn.StopUnitContext(context.Background(), quadletService, "replace", ch)
	if err != nil {
		return fmt.Errorf("error stopping unit %s: %w", quadletService, err)
	}
	if config.GetConfig().Verbose {
		log.Printf("successfully stopped unit %s\n", quadletService)
	}
	return nil
}

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

func ShowUnit(unitName string, unitType string) error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()
	quadletService := unitName
	if unitType == "container" {
		quadletService += ".service"
	} else {
		quadletService += "-" + unitType + ".service"
	}
	prop, err := conn.GetUnitPropertiesContext(ctx, quadletService)
	if err != nil {
		return fmt.Errorf("error getting unit properties: %w", err)
	}

	fmt.Printf("\n=== %s ===\n\n", quadletService)

	fmt.Println("Status:")
	fmt.Printf("  %-20s: %v\n", "State", prop["ActiveState"])
	fmt.Printf("  %-20s: %v\n", "Sub-State", prop["SubState"])
	fmt.Printf("  %-20s: %v\n", "Load State", prop["LoadState"])

	fmt.Println("\nUnit Information:")
	fmt.Printf("  %-20s: %v\n", "Description", prop["Description"])
	fmt.Printf("  %-20s: %v\n", "Path", prop["FragmentPath"])

	// Read and display the actual quadlet configuration
	if fragmentPath, ok := prop["FragmentPath"].(string); ok {
		content, err := os.ReadFile(fragmentPath)
		if err == nil {
			unitConfig, _ := ini.Load(content)
			sectionName := fmt.Sprintf("X-%s", strings.Title(unitType))
			if section, err := unitConfig.GetSection(sectionName); err == nil {
				fmt.Printf("\n%s Configuration:\n", strings.Title(unitType))
				for _, key := range section.Keys() {
					fmt.Printf("  %-20s: %s\n", key.Name(), key.Value())
				}
			}
		}
	}

	fmt.Println()
	return nil
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
