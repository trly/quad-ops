package main

import (
	"context"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
)

// UpdateCmd represents the update command.
type UpdateCmd struct{}

// Run executes the update command.
func (u *UpdateCmd) Run() error {
	fmt.Printf("Current version: %s\n", Version)

	if Version == "dev" {
		fmt.Println("Update check skipped for dev version")
		return nil
	}

	fmt.Println("Checking for updates...")

	// Detect latest version
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug("trly/quad-ops"))
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !found {
		fmt.Println("No release found")
		return nil
	}

	if latest.LessOrEqual(Version) {
		fmt.Println("You are already running the latest version.")
		return nil
	}

	fmt.Printf("Update available! New version: %s\n", latest.Version())
	fmt.Println("Downloading and applying update...")

	// Get current executable path
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Update to the latest version
	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Update completed successfully! Please restart %s to use the new version.\n", "quad-ops")
	return nil
}
