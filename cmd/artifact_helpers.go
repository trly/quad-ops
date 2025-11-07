/*
Copyright © 2025 Travis Lyons travis.lyons@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package cmd provides artifact helper functions for cross-platform support
package cmd

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/platform/launchd"
)

// parseServiceNameFromArtifact extracts the service name from an artifact path
// in a platform-neutral way.
//
// For systemd/quadlet artifacts (.container, .network, .volume, .build):
//   - "myapp-web.container" -> "myapp-web"
//   - "myapp-db-volume.volume" -> "myapp-db-volume" (preserves '-volume' in name)
//   - "api.container" -> "api"
//
// For launchd artifacts (.plist):
//   - "com.example.web-service.plist" -> "web-service"
//   - "dev.trly.quad-ops.api.plist" -> "api"
//   - "simple.plist" -> "simple"
func parseServiceNameFromArtifact(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	if ext == ".plist" {
		// For launchd plists, extract the service name after the last dot
		// Label format is typically: <prefix>.<serviceName>
		// e.g., "com.example.web-service" -> "web-service"
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			return name[idx+1:]
		}
		return name
	}

	// For systemd/quadlet, the base name (without extension) IS the unit name
	// Do NOT strip type suffixes - they may be part of the actual unit name
	// e.g., "myapp-db-volume.volume" has unit name "myapp-db-volume"
	return name
}

// isServiceArtifact determines if an artifact represents a service
// that can be started/stopped (as opposed to networks, volumes, etc.).
//
// Returns true for:
//   - .container files (systemd/quadlet)
//   - .plist files (launchd)
func isServiceArtifact(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".container" || ext == ".plist"
}

// matchesServiceName checks if an artifact path matches the given service name.
// Handles both systemd and launchd naming conventions.
//
// For systemd: direct base name match
//   - "web-service.container" matches "web-service"
//
// For launchd: suffix-based match for labels
//   - "com.example.web-service.plist" matches "web-service"
func matchesServiceName(artifactPath, serviceName string) bool {
	base := filepath.Base(artifactPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Direct match
	if name == serviceName {
		return true
	}

	// For .plist files, check if the name ends with ".<serviceName>"
	if ext == ".plist" && strings.HasSuffix(name, "."+serviceName) {
		return true
	}

	return false
}

// allowedQuadletExt defines the set of valid quadlet artifact extensions.
var allowedQuadletExt = map[string]struct{}{
	".container": {},
	".service":   {},
	".network":   {},
	".volume":    {},
	".target":    {},
	".timer":     {},
	".build":     {},
}

// filterArtifactsForPlatform filters artifacts based on platform-specific rules.
// On macOS, it filters launchd plists to only include those with the configured label prefix.
// On Linux, it filters to only include valid quadlet unit file extensions.
func filterArtifactsForPlatform(artifacts []platform.Artifact, cfg *config.Settings) []platform.Artifact {
	if runtime.GOOS == "darwin" {
		opts := launchd.OptionsFromSettings(cfg.RepositoryDir, cfg.QuadletDir, cfg.UserMode)
		return filterLaunchdArtifacts(artifacts, opts.LabelPrefix)
	}
	return filterQuadletArtifacts(artifacts)
}

// filterLaunchdArtifacts filters artifacts to only include .plist files with the given label prefix.
func filterLaunchdArtifacts(artifacts []platform.Artifact, labelPrefix string) []platform.Artifact {
	var filtered []platform.Artifact
	for _, artifact := range artifacts {
		ext := filepath.Ext(artifact.Path)
		if ext != ".plist" {
			continue
		}

		base := filepath.Base(artifact.Path)
		name := strings.TrimSuffix(base, ext)

		if strings.HasPrefix(name, labelPrefix) {
			filtered = append(filtered, artifact)
		}
	}
	return filtered
}

// filterQuadletArtifacts filters artifacts to only include valid quadlet unit file extensions.
func filterQuadletArtifacts(artifacts []platform.Artifact) []platform.Artifact {
	var filtered []platform.Artifact
	for _, artifact := range artifacts {
		ext := filepath.Ext(artifact.Path)
		if _, ok := allowedQuadletExt[ext]; ok {
			filtered = append(filtered, artifact)
		}
	}
	return filtered
}
