package buildinfo

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/creativeprojects/go-selfupdate"
)

var (
	// Commit is the VCS revision hash.
	Commit = "none"
	// Date is the VCS commit timestamp.
	Date = "unknown"
	// GoVersion is the Go toolchain version used to build the binary.
	GoVersion = "unknown"
	// Version is the release version, set via ldflags by goreleaser.
	Version = "dev"
)

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	GoVersion = bi.GoVersion

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			Commit = s.Value
		case "vcs.time":
			Date = s.Value
		}
	}
}

// IsDev reports whether this is a development build.
func IsDev() bool {
	return Version == "dev"
}

// UpdateStatus represents the result of an update check.
type UpdateStatus struct {
	Available  bool
	NewVersion string
	AssetURL   string
	AssetName  string
}

// CheckForUpdates checks GitHub for a newer release.
// Returns an UpdateStatus and any error encountered.
func CheckForUpdates(ctx context.Context) (*UpdateStatus, error) {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("trly/quad-ops"))
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	if !found {
		return &UpdateStatus{}, nil
	}

	if latest.LessOrEqual(Version) {
		return &UpdateStatus{}, nil
	}

	return &UpdateStatus{
		Available:  true,
		NewVersion: latest.Version(),
		AssetURL:   latest.AssetURL,
		AssetName:  latest.AssetName,
	}, nil
}
