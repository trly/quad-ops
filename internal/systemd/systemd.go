package systemd

import (
	"context"
	"fmt"
	"log"
	"quad-ops/internal/config"

	"github.com/coreos/go-systemd/v22/dbus"
)

func ReloadAndRestartUnit(cfg config.Config, unitName string, unitType string) error {
	var conn *dbus.Conn
	var err error
	var quadletService string
	if unitType == "container" {
		quadletService = unitName + ".service"
	} else {
		quadletService = unitName + "-" + unitType + ".service"
	}
	ctx := context.Background()

	if cfg.Verbose {
		log.Printf("initializing systemd connection for unit %s (user mode: %v)\n", quadletService, cfg.UserMode)
	}

	if cfg.UserMode {
		if cfg.Verbose {
			log.Printf("establishing user connection to systemd...")
		}
		conn, err = dbus.NewUserConnectionContext(ctx)
	} else {
		if cfg.Verbose {
			log.Printf("establishing system connection to systemd...")
		}
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}

	if err != nil {
		return fmt.Errorf("connecting to systemd: %w", err)
	}
	defer conn.Close()
	if cfg.Verbose {
		log.Printf("successfully connected to systemd")
	}

	if cfg.Verbose {
		log.Printf("reloading systemd daemon...")
	}
	err = conn.ReloadContext(context.Background())
	if err != nil {
		return fmt.Errorf("reloading systemd: %w", err)
	}
	if cfg.Verbose {
		log.Printf("successfully reloaded systemd daemon")
	}

	if cfg.Verbose {
		log.Printf("initiating restart of unit %s...\n", quadletService)
	}
	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), quadletService, "replace", ch)
	if err != nil {
		return fmt.Errorf("restarting unit %s: %w", quadletService, err)
	}

	if cfg.Verbose {
		log.Printf("waiting for unit restart to complete...")
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
