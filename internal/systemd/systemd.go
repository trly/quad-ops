package systemd

import (
	"context"
	"fmt"
	"log"

	"github.com/trly/quad-ops/internal/config"

	"github.com/coreos/go-systemd/v22/dbus"
)

var ctx = context.Background()

func GetUnitStatus(cfg config.Config, unitName string, unitType string) (string, error) {
	conn, err := getSystemdConnection(cfg)
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

func RestartUnit(cfg config.Config, unitName string, unitType string) error {
	conn, err := getSystemdConnection(cfg)
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
	if cfg.Verbose {
		log.Printf("successfully restarted unit %s\n", quadletService)
	}

	return nil
}

func StartUnit(cfg config.Config, unitName string, unitType string) error {
	conn, err := getSystemdConnection(cfg)
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
	if cfg.Verbose {
		log.Printf("successfully started unit %s\n", quadletService)
	}
	return nil
}

func StopUnit(cfg config.Config, unitName string, unitType string) error {
	conn, err := getSystemdConnection(cfg)
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
	if cfg.Verbose {
		log.Printf("successfully stopped unit %s\n", quadletService)
	}
	return nil
}

func ReloadSystemd(cfg config.Config) error {
	conn, err := getSystemdConnection(cfg)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	if cfg.Verbose {
		log.Printf("reloading systemd...")
	}
	err = conn.ReloadContext(ctx)
	if err != nil {
		return fmt.Errorf("error reloading systemd: %w", err)
	}

	return nil
}

func getSystemdConnection(cfg config.Config) (*dbus.Conn, error) {
	if cfg.Verbose {
		if cfg.UserMode {
			log.Printf("establishing user connection to systemd...")
		} else {
			log.Printf("establishing system connection to systemd...")
		}
	}

	if cfg.UserMode {
		return dbus.NewUserConnectionContext(ctx)
	}
	return dbus.NewSystemConnectionContext(ctx)
}
