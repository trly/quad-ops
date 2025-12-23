package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = time.Now().Format(time.RFC3339)
)

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Printf("quad-ops version %s\n", Version)
	fmt.Printf("  commit: %s\n", Commit)
	fmt.Printf("  built: %s\n", Date)
	fmt.Printf("  go: %s\n", runtime.Version())

	v.checkForUpdates()

	return nil
}

func (v *VersionCmd) checkForUpdates() {
	if Version == "dev" {
		fmt.Println("\nSkipping update check for development build")
		return
	}

	fmt.Println("\nChecking for updates...")

	// Detect latest version
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug("trly/quad-ops"))
	if err != nil {
		fmt.Printf("Failed to check for updates: %v\n", err)
		return
	}

	if !found {
		fmt.Println("No release found")
		return
	}

	if latest.LessOrEqual(Version) {
		fmt.Println("You are using the latest version.")
		return
	}

	fmt.Printf("Update available! New version: %s\n", latest.Version())
	fmt.Println("Run 'quad-ops update' to update to the latest version.")
}
