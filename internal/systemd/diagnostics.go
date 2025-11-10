package systemd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/trly/quad-ops/internal/log"
)

// FileSystemChecker provides file system operations for diagnostics.
type FileSystemChecker interface {
	Stat(path string) (bool, error)
}

// DiagnosticIssue represents a detected problem with the Quadlet generator.
type DiagnosticIssue struct {
	Type        string   // Type of issue: "generator_missing", "unit_not_generated", etc.
	Message     string   // Human-readable description
	Suggestions []string // Actionable recommendations
}

// CheckGeneratorBinaryExists verifies that the Quadlet generator binary is installed.
func CheckGeneratorBinaryExists(generatorPath string, fs FileSystemChecker, logger log.Logger) (bool, error) {
	logger.Debug("Checking for generator binary", "path", generatorPath)

	exists, err := fs.Stat(generatorPath)
	if err != nil {
		logger.Error("Error checking generator binary", "path", generatorPath, "error", err)
		return false, fmt.Errorf("error checking generator binary at %s: %w", generatorPath, err)
	}

	if !exists {
		logger.Warn("Generator binary not found", "path", generatorPath)
		return false, nil
	}

	logger.Debug("Generator binary found", "path", generatorPath)
	return true, nil
}

// CheckUnitLoaded verifies that a systemd unit is loaded.
func CheckUnitLoaded(ctx context.Context, unitName string, factory ConnectionFactory, userMode bool, logger log.Logger) (bool, error) {
	logger.Debug("Checking if unit is loaded", "unit", unitName)

	conn, err := factory.NewConnection(ctx, userMode)
	if err != nil {
		logger.Error("Failed to connect to systemd", "error", err)
		return false, fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	props, err := conn.GetUnitProperties(ctx, unitName)
	if err != nil {
		// Unit not found or not loaded
		logger.Debug("Unit not loaded", "unit", unitName)
		return false, nil
	}

	// Check LoadState property
	if loadState, ok := props["LoadState"].(string); ok {
		loaded := loadState == "loaded"
		logger.Debug("Unit load state", "unit", unitName, "loadState", loadState, "loaded", loaded)
		return loaded, nil
	}

	logger.Debug("Unit properties retrieved but LoadState missing", "unit", unitName)
	return false, nil
}

// DiagnoseGeneratorIssues performs comprehensive diagnostics to identify why units may not be available.
func DiagnoseGeneratorIssues(
	ctx context.Context,
	generatorPath string,
	artifacts []string,
	fs FileSystemChecker,
	factory ConnectionFactory,
	userMode bool,
	logger log.Logger,
) []DiagnosticIssue {
	var issues []DiagnosticIssue

	// Check 1: Is the generator binary installed?
	generatorExists, err := CheckGeneratorBinaryExists(generatorPath, fs, logger)
	if err != nil {
		logger.Error("Error checking generator binary", "error", err)
		// Continue diagnostics even if we can't check the binary
	} else if !generatorExists {
		issues = append(issues, DiagnosticIssue{
			Type:    "generator_missing",
			Message: fmt.Sprintf("Quadlet generator binary not found at %s", generatorPath),
			Suggestions: []string{
				"Install podman (the generator is included with podman)",
				"Verify podman version >= 4.4 (Quadlet was introduced in 4.4)",
				"For Fedora/RHEL: sudo dnf install podman",
				"For Ubuntu: sudo apt install podman",
				"For macOS: brew install podman",
				fmt.Sprintf("Verify generator exists: ls -la %s", generatorPath),
			},
		})
		// If generator is missing, no point checking units
		return issues
	}

	// Check 2: For each artifact, verify the corresponding unit is loaded
	for _, artifactPath := range artifacts {
		unitName := ArtifactPathToUnitName(artifactPath)
		loaded, err := CheckUnitLoaded(ctx, unitName, factory, userMode, logger)
		if err != nil {
			logger.Error("Error checking unit load state", "unit", unitName, "artifact", artifactPath, "error", err)
			continue
		}

		if !loaded {
			artifactName := filepath.Base(artifactPath)
			issues = append(issues, DiagnosticIssue{
				Type:    "unit_not_generated",
				Message: fmt.Sprintf("%s exists but %s not loaded in systemd", artifactName, unitName),
				Suggestions: []string{
					"Run: systemctl daemon-reload (or systemctl --user daemon-reload for user mode)",
					fmt.Sprintf("Check generator logs: journalctl -u systemd-system-generators.target -n 50"),
					fmt.Sprintf("Verify artifact syntax: cat %s", artifactPath),
					"Generator may have failed silently - check for syntax errors in .container/.network/.volume files",
					fmt.Sprintf("Try manually: /usr/lib/systemd/system-generators/podman-system-generator /tmp/test"),
				},
			})
		}
	}

	return issues
}

// ArtifactPathToUnitName converts a Quadlet artifact path to the expected systemd unit name.
// Examples:
//   - /etc/containers/systemd/test.container → test.service
//   - /etc/containers/systemd/mynet.network → mynet-network.service
//   - /etc/containers/systemd/myvol.volume → myvol-volume.service
func ArtifactPathToUnitName(artifactPath string) string {
	base := filepath.Base(artifactPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Map Quadlet artifact types to systemd unit names
	switch ext {
	case ".container":
		// .container files generate .service units with the same base name
		return nameWithoutExt + ".service"
	case ".network":
		// .network files generate <name>-network.service units
		return nameWithoutExt + "-network.service"
	case ".volume":
		// .volume files generate <name>-volume.service units
		return nameWithoutExt + "-volume.service"
	case ".build":
		// .build files generate .service units with the same base name
		return nameWithoutExt + ".service"
	case ".image":
		// .image files generate .service units with the same base name
		return nameWithoutExt + ".service"
	case ".pod":
		// .pod files generate .service units with the same base name
		return nameWithoutExt + ".service"
	case ".kube":
		// .kube files generate .service units with the same base name
		return nameWithoutExt + ".service"
	default:
		// Unknown artifact type, assume .service
		return nameWithoutExt + ".service"
	}
}

// FormatDiagnosticIssue formats a diagnostic issue for display.
func FormatDiagnosticIssue(issue DiagnosticIssue) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Issue: %s\n", issue.Message))

	if len(issue.Suggestions) > 0 {
		output.WriteString("\nSuggestions:\n")
		for _, suggestion := range issue.Suggestions {
			output.WriteString(fmt.Sprintf("  - %s\n", suggestion))
		}
	}

	return output.String()
}
