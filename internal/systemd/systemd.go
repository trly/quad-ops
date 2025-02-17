package systemd

import (
	"context"
	"fmt"
	"log"

	"github.com/coreos/go-systemd/v22/dbus"
)

func ReloadAndRestartUnit(unitName string, userMode bool, verbose bool) error {
	var conn *dbus.Conn
	var err error
	var quadletService = unitName + ".service"
	ctx := context.Background()

	if verbose {
		log.Printf("Initializing systemd connection for unit %s (user mode: %v)\n", quadletService, userMode)
	}

	if userMode {
		if verbose {
			log.Printf("Establishing user connection to systemd...")
		}
		conn, err = dbus.NewUserConnectionContext(ctx)
	} else {
		if verbose {
			log.Printf("Establishing system connection to systemd...")
		}
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}

	if err != nil {
		return fmt.Errorf("connecting to systemd: %w", err)
	}
	defer conn.Close()
	if verbose {
		log.Printf("Successfully connected to systemd")
	}

	if verbose {
		log.Printf("Reloading systemd daemon...")
	}
	err = conn.ReloadContext(context.Background())
	if err != nil {
		return fmt.Errorf("reloading systemd: %w", err)
	}
	if verbose {
		log.Printf("Successfully reloaded systemd daemon")
	}

	if verbose {
		log.Printf("Initiating restart of unit %s...\n", quadletService)
	}
	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), quadletService, "replace", ch)
	if err != nil {
		return fmt.Errorf("restarting unit %s: %w", quadletService, err)
	}

	if verbose {
		log.Printf("Waiting for unit restart to complete...")
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("unit restart failed: %s", result)
	}
	if verbose {
		log.Printf("Successfully restarted unit %s\n", quadletService)
	}

	return nil
}
