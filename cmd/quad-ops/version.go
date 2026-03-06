package main

import (
	"context"
	"fmt"

	"github.com/trly/quad-ops/internal/buildinfo"
)

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Printf("quad-ops version %s\n", buildinfo.Version)
	fmt.Printf("  commit: %s\n", buildinfo.Commit)
	fmt.Printf("  built: %s\n", buildinfo.Date)
	fmt.Printf("  go: %s\n", buildinfo.GoVersion)

	if buildinfo.IsDev() {
		fmt.Println("\nSkipping update check for development build")
		return nil
	}

	fmt.Println("\nChecking for updates...")

	status, err := buildinfo.CheckForUpdates(context.Background())
	if err != nil {
		fmt.Printf("Failed to check for updates: %v\n", err)
		return nil
	}

	if !status.Available {
		fmt.Println("You are using the latest version.")
		return nil
	}

	fmt.Printf("Update available! New version: %s\n", status.NewVersion)
	fmt.Println("Run 'quad-ops update' to update to the latest version.")

	return nil
}
