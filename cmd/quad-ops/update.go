package main

import (
	"context"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/trly/quad-ops/internal/buildinfo"
)

// UpdateCmd represents the update command.
type UpdateCmd struct{}

// Run executes the update command.
func (u *UpdateCmd) Run() error {
	fmt.Printf("Current version: %s\n", buildinfo.Version)

	if buildinfo.IsDev() {
		fmt.Println("Update check skipped for dev version")
		return nil
	}

	fmt.Println("Checking for updates...")

	status, err := buildinfo.CheckForUpdates(context.Background())
	if err != nil {
		return err
	}

	if !status.Available {
		fmt.Println("You are already running the latest version.")
		return nil
	}

	fmt.Printf("Update available! New version: %s\n", status.NewVersion)
	fmt.Println("Downloading and applying update...")

	// Get current executable path
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Update to the latest version
	if err := selfupdate.UpdateTo(context.Background(), status.AssetURL, status.AssetName, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Update completed successfully! Please restart %s to use the new version.\n", "quad-ops")
	return nil
}
