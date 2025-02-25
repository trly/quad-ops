package systemd

import (
	"context"
	"fmt"
	"log"

	"github.com/coreos/go-systemd/v22/dbus"
)

func ReloadAndRestartUnit(unitName string, unitType string, userMode bool, verbose bool) error {
	var conn *dbus.Conn
	var err error
	var quadletService string
	if unitType == "container" {
		quadletService = unitName + ".service"
	} else {
		quadletService = unitName + "-" + unitType + ".service"
	}
	ctx := context.Background()

	if verbose {
		log.Printf("initializing systemd connection for unit %s (user mode: %v)\n", quadletService, userMode)
	}

	if userMode {
		if verbose {
			log.Printf("establishing user connection to systemd...")
		}
		conn, err = dbus.NewUserConnectionContext(ctx)
	} else {
		if verbose {
			log.Printf("establishing system connection to systemd...")
		}
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}

	if err != nil {
		return fmt.Errorf("connecting to systemd: %w", err)
	}
	defer conn.Close()
	if verbose {
		log.Printf("successfully connected to systemd")
	}

	if verbose {
		log.Printf("reloading systemd daemon...")
	}
	err = conn.ReloadContext(context.Background())
	if err != nil {
		return fmt.Errorf("reloading systemd: %w", err)
	}
	if verbose {
		log.Printf("successfully reloaded systemd daemon")
	}

	if verbose {
		log.Printf("initiating restart of unit %s...\n", quadletService)
	}
	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), quadletService, "replace", ch)
	if err != nil {
		return fmt.Errorf("restarting unit %s: %w", quadletService, err)
	}

	if verbose {
		log.Printf("waiting for unit restart to complete...")
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("unit restart failed: %s", result)
	}
	if verbose {
		log.Printf("successfully restarted unit %s\n", quadletService)
	}

	return nil
}
