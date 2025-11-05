//go:build darwin

package launchd

import (
	"os"
	"path/filepath"
)

// testOptions returns test options with mock podman path.
// This is shared across all test files in the launchd package.
func testOptions() Options {
	// Create a temporary file to act as mock podman binary
	tmpDir := os.TempDir()
	mockPodman := filepath.Join(tmpDir, "podman-mock")

	// Create the mock file if it doesn't exist
	_ = os.WriteFile(mockPodman, []byte("#!/bin/sh\n"), 0600)

	// Use temp directories for tests (writable)
	plistDir := filepath.Join(tmpDir, "LaunchAgents")
	logsDir := filepath.Join(tmpDir, "Logs", "quad-ops")

	return Options{
		Domain:      DomainUser,
		PodmanPath:  mockPodman,
		LabelPrefix: "dev.trly.quad-ops",
		PlistDir:    plistDir,
		LogsDir:     logsDir,
		UID:         501,
		UseSudo:     false,
	}
}
