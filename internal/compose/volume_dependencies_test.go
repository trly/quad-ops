package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

// TestVolumeDependencies_ServiceWithExplicitVolumes tests that a service
// with explicit volume mounts only gets dependencies on those volumes.
func TestVolumeDependencies_ServiceWithExplicitVolumes(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"data": {
				Name:   "data",
				Driver: "local",
			},
			"logs": {
				Name:   "logs",
				Driver: "local",
			},
			"cache": {
				Name:   "cache",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "data",
						Target: "/data",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "myapp-web", spec.Name)

	// CRITICAL: Service should only have volumes it actually uses, not all project volumes
	assert.Len(t, spec.Volumes, 1, "Service should only depend on volumes it mounts")
	assert.Equal(t, "myapp-data", spec.Volumes[0].Name)
}

// TestVolumeDependencies_ServiceWithMultipleVolumes tests that a service
// with multiple volume mounts gets dependencies on all of them.
func TestVolumeDependencies_ServiceWithMultipleVolumes(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"data": {
				Name:   "data",
				Driver: "local",
			},
			"logs": {
				Name:   "logs",
				Driver: "local",
			},
			"cache": {
				Name:   "cache",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"app": {
				Name:  "app",
				Image: "app:1.0",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "data",
						Target: "/data",
					},
					{
						Type:   "volume",
						Source: "logs",
						Target: "/var/log",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Should have exactly the volumes it uses
	assert.Len(t, spec.Volumes, 2)
	volumeNames := []string{spec.Volumes[0].Name, spec.Volumes[1].Name}
	assert.ElementsMatch(t, []string{"myapp-data", "myapp-logs"}, volumeNames)
}

// TestVolumeDependencies_ServiceWithNoVolumes tests that a service
// with no volume mounts has no volume dependencies.
func TestVolumeDependencies_ServiceWithNoVolumes(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"data": {
				Name:   "data",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:    "web",
				Image:   "nginx:latest",
				Volumes: nil,
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Service without volume mounts should have no volume dependencies
	assert.Empty(t, spec.Volumes, "Service with no mounts should not depend on project volumes")
}

// TestVolumeDependencies_ServiceWithBindMountsOnly tests that a service
// with only bind mounts (no named volumes) has no volume dependencies.
func TestVolumeDependencies_ServiceWithBindMountsOnly(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"data": {
				Name:   "data",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "bind",
						Source: "/host/path",
						Target: "/container/path",
					},
					{
						Type:   "bind",
						Source: "./relative",
						Target: "/data",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Bind mounts don't create volume unit dependencies
	assert.Empty(t, spec.Volumes, "Service with only bind mounts should have no volume dependencies")

	// But should still have the mounts in Container.Mounts
	assert.Len(t, spec.Container.Mounts, 2)
	assert.Equal(t, service.MountTypeBind, spec.Container.Mounts[0].Type)
	assert.Equal(t, service.MountTypeBind, spec.Container.Mounts[1].Type)
}

// TestVolumeDependencies_ServiceWithMixedMounts tests that a service
// with both bind mounts and named volumes only gets dependencies on named volumes.
func TestVolumeDependencies_ServiceWithMixedMounts(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"data": {
				Name:   "data",
				Driver: "local",
			},
			"logs": {
				Name:   "logs",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"app": {
				Name:  "app",
				Image: "app:1.0",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "bind",
						Source: "/host/config",
						Target: "/config",
					},
					{
						Type:   "volume",
						Source: "data",
						Target: "/data",
					},
					{
						Type:   "tmpfs",
						Target: "/tmp",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Should only depend on named volumes, not bind mounts or tmpfs
	assert.Len(t, spec.Volumes, 1)
	assert.Equal(t, "myapp-data", spec.Volumes[0].Name)

	// All mounts should still be in Container.Mounts
	assert.Len(t, spec.Container.Mounts, 3)
}

// TestVolumeDependencies_ExternalVolumes tests that external volumes
// are included in spec.Volumes and properly marked as external.
func TestVolumeDependencies_ExternalVolumes(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"shared-data": {
				Name:     "shared-data",
				External: types.External(true),
			},
			"local-data": {
				Name:   "local-data",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"app": {
				Name:  "app",
				Image: "app:1.0",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "shared-data",
						Target: "/shared",
					},
					{
						Type:   "volume",
						Source: "local-data",
						Target: "/local",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Should have both volumes
	assert.Len(t, spec.Volumes, 2)

	// Find the external volume
	var externalVol *service.Volume
	var localVol *service.Volume
	for i := range spec.Volumes {
		if spec.Volumes[i].External {
			externalVol = &spec.Volumes[i]
		} else {
			localVol = &spec.Volumes[i]
		}
	}

	require.NotNil(t, externalVol, "Should have external volume")
	require.NotNil(t, localVol, "Should have local volume")

	// External volume should NOT be prefixed
	assert.Equal(t, "shared-data", externalVol.Name)
	assert.True(t, externalVol.External)

	// Local volume should be prefixed
	assert.Equal(t, "myapp-local-data", localVol.Name)
	assert.False(t, localVol.External)
}

// TestVolumeDependencies_MultipleServicesShareVolume tests that multiple
// services using the same volume each get their own dependency on it.
func TestVolumeDependencies_MultipleServicesShareVolume(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"shared": {
				Name:   "shared",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "shared",
						Target: "/data",
					},
				},
			},
			"worker": {
				Name:  "worker",
				Image: "worker:1.0",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "shared",
						Target: "/data",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 2)

	// Both services should have the shared volume
	for _, spec := range specs {
		assert.Len(t, spec.Volumes, 1)
		assert.Equal(t, "myapp-shared", spec.Volumes[0].Name)
	}
}

// TestVolumeDependencies_NoProjectVolumes tests behavior when project
// has no volumes defined but service has bind mounts.
func TestVolumeDependencies_NoProjectVolumes(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name:    "myapp",
		Volumes: map[string]types.VolumeConfig{},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "bind",
						Source: "/host/data",
						Target: "/data",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// No project volumes and only bind mounts = no volume dependencies
	assert.Empty(t, spec.Volumes)
	assert.Len(t, spec.Container.Mounts, 1)
}

// TestVolumeDependencies_AutoDetectedVolumes tests that auto-detected
// volumes (without explicit type) are handled correctly.
func TestVolumeDependencies_AutoDetectedVolumes(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Volumes: map[string]types.VolumeConfig{
			"data": {
				Name:   "data",
				Driver: "local",
			},
		},
		Services: map[string]types.ServiceConfig{
			"app": {
				Name:  "app",
				Image: "app:1.0",
				Volumes: []types.ServiceVolumeConfig{
					{
						// No explicit type - should auto-detect
						Source: "data",
						Target: "/data",
					},
					{
						// No explicit type - should auto-detect as bind
						Source: "/absolute/path",
						Target: "/host",
					},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Should have the named volume (data) but not the bind mount
	assert.Len(t, spec.Volumes, 1)
	assert.Equal(t, "myapp-data", spec.Volumes[0].Name)

	// Both should be in mounts
	assert.Len(t, spec.Container.Mounts, 2)
}
