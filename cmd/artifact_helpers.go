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
	"strings"
)

// parseServiceNameFromArtifact extracts the service name from an artifact path
// in a platform-neutral way.
//
// For systemd/quadlet artifacts (.container, .network, .volume, .build):
//   - "web-service-container.container" -> "web-service"
//   - "web-service-network.network" -> "web-service"
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

	// For systemd/quadlet, strip common suffixes
	suffixes := []string{"-container", "-network", "-volume", "-build"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}

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
