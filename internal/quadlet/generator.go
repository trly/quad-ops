package quadlet

import (
	"fmt"
	"log"
	"strings"
)

func GenerateQuadletUnit(unit QuadletUnit, verbose bool) string {
	if verbose {
		log.Printf("Generating Quadlet unit for %s of type %s", unit.Name, unit.Type)
	}

	// [Unit] section
	content := fmt.Sprintf("[Unit]\nDescription=%s\n", unit.Systemd.Description)
	if len(unit.Systemd.After) > 0 {
		if verbose {
			log.Printf("Adding After dependencies: %v", unit.Systemd.After)
		}
		content += fmt.Sprintf("After=%s\n", strings.Join(unit.Systemd.After, " "))
	}

	// Type-specific sections
	switch unit.Type {
	case "container":
		content += "\n[Container]\n"
		if unit.Container.Image != "" {
			content += fmt.Sprintf("Image=%s\n", unit.Container.Image)
		}
		for _, label := range unit.Container.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
		for _, publishPort := range unit.Container.PublishPort {
			content += fmt.Sprintf("PublishPort=%s\n", publishPort)
		}
	case "volume":
		content += "\n[Volume]\n"
		for _, label := range unit.Volume.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
	case "network":
		content += "\n[Network]\n"
		for _, label := range unit.Network.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
	case "pod":
		content += "\n[Pod]\n"
		for _, label := range unit.Pod.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
	case "kube":
		content += "\n[Kube]\n"
		if unit.Kube.Path != "" {
			content += fmt.Sprintf("Path=%s\n", unit.Kube.Path)
		}
	case "image":
		content += "\n[Image]\n"
		if unit.Image.Image != "" {
			content += fmt.Sprintf("Image=%s\n", unit.Image.Image)
		}
	case "build":
		content += "\n[Build]\n"
		if unit.Build.Context != "" {
			content += fmt.Sprintf("Context=%s\n", unit.Build.Context)
		}
		if unit.Build.Dockerfile != "" {
			content += fmt.Sprintf("Dockerfile=%s\n", unit.Build.Dockerfile)
		}
	}

	// [Service] section
	content += "\n[Service]\n"
	if unit.Systemd.RestartPolicy != "" {
		if verbose {
			log.Printf("Setting restart policy to: %s", unit.Systemd.RestartPolicy)
		}
		content += fmt.Sprintf("Restart=%s\n", unit.Systemd.RestartPolicy)
	}

	return content
}
