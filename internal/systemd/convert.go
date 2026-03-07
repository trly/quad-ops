package systemd

import (
	"fmt"
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
// The repo parameter provides repository metadata for fixed labels applied to all units.
func Convert(project *types.Project, repo RepositoryMeta) ([]Unit, error) {
	units := make([]Unit, 0)
	projectName := project.Name

	// Convert volumes (skip external volumes - they reference existing volumes)
	for volName, vol := range project.Volumes {
		if vol.External {
			continue
		}
		units = append(units, BuildVolume(projectName, volName, &vol, repo))
	}

	// Convert networks (skip external networks - they reference existing networks)
	for netName, net := range project.Networks {
		if net.External {
			continue
		}
		units = append(units, BuildNetwork(projectName, netName, &net, repo))
	}

	// Convert services
	for svcName, svc := range project.Services {
		resolveBindMountPaths(&svc, project.WorkingDir)
		units = append(units, BuildContainer(projectName, svcName, &svc, project.Networks, project.Volumes, repo))
	}

	return units, nil
}

// effectiveName returns the Podman resource name to use. If the compose config
// has an explicit name that differs from compose-go's auto-generated
// "{project}_{resource}" default, that explicit name takes priority. Otherwise
// the dash-separated unitBaseName is used so the Podman name matches the
// Quadlet unit file name.
func effectiveName(composeName, projectName, resourceName, unitBaseName string) string {
	if composeName != "" && composeName != fmt.Sprintf("%s_%s", projectName, resourceName) {
		return composeName
	}
	return unitBaseName
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
