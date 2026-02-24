package systemd

import (
	"path/filepath"

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
		resolveBindMountPaths(&svc, project.WorkingDir)
		units = append(units, BuildContainer(projectName, svcName, &svc, project.Networks))
	}

	return units, nil
}

// resolveBindMountPaths resolves relative source paths in bind mount volumes
// to absolute paths using the project's working directory.
func resolveBindMountPaths(svc *types.ServiceConfig, workingDir string) {
	if workingDir == "" {
		return
	}
	for i, vol := range svc.Volumes {
		if vol.Type != types.VolumeTypeBind {
			continue
		}
		if vol.Source != "" && !filepath.IsAbs(vol.Source) {
			svc.Volumes[i].Source = filepath.Join(workingDir, vol.Source)
		}
	}
}
