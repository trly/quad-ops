// Package podman containers podman stuff.
package podman

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/podman/v5/pkg/bindings"
)

// GetConnection returns a connection to the Podman API.
func GetConnection() context.Context {
	conn, err := bindings.NewConnection(context.Background(), "unix:///run/user/1000/podman/podman.sock")
	if err != nil {
		fmt.Println("Error connecting to Podman API:", err)
		os.Exit(1)
	}

	return conn
}
