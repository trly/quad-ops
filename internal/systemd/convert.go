package systemd

import (
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/ini.v1"
)

// Unit represents a systemd unit file.
type Unit struct {
	Name string
	File *ini.File
}

// Convert transforms a loaded compose project into systemd unit files.
func Convert(project *types.Project) ([]Unit, error) {
	units := make([]Unit, 0)
	projectName := project.Name

	// Convert volumes (skip external volumes - they reference existing volumes)
	for volName, vol := range project.Volumes {
		if vol.External {
			continue
		}
		units = append(units, BuildVolume(projectName, volName, &vol))
	}

	// Convert networks (skip external networks - they reference existing networks)
	for netName, net := range project.Networks {
		if net.External {
			continue
		}
		units = append(units, BuildNetwork(projectName, netName, &net))
	}

	// Convert services
	for svcName, svc := range project.Services {
		units = append(units, BuildContainer(projectName, svcName, &svc))
	}

	return units, nil
}
