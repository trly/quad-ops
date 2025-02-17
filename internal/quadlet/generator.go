package quadlet

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// GenerateQuadletUnit creates a systemd unit file content from a QuadletUnit struct.
// It includes Unit, type-specific, Service, and Install sections with the provided configuration.
func GenerateQuadletUnit(unit QuadletUnit) string {
	content := fmt.Sprintf("[Unit]\nDescription=%s\n", unit.Systemd.Description)
	if len(unit.Systemd.After) > 0 {
		content += fmt.Sprintf("After=%s\n", strings.Join(unit.Systemd.After, " "))
	}

	// Type-specific section
	sectionName := cases.Title(language.Und).String(unit.Type)
	content += fmt.Sprintf("\n[%s]\n", sectionName)
	for key, value := range unit.Config {
		content += fmt.Sprintf("%s=%s\n", key, value)
	}

	// Service section
	content += "\n[Service]\n"
	if unit.Systemd.RestartPolicy != "" {
		content += fmt.Sprintf("Restart=%s\n", unit.Systemd.RestartPolicy)
	}

	// Install section
	content += "\n[Install]\nWantedBy=multi-user.target\n"

	return content
}
