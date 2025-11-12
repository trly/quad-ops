package compose

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

// ---------------------------
// ConvertProject edge cases
// ---------------------------

func TestConverter_ConvertProject_BasicService(t *testing.T) {
	project := &types.Project{
		Name:       "test",
		WorkingDir: "/test",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
			},
		},
	}

	converter := NewConverter("/test")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "test-web", spec.Name)
	assert.Equal(t, "nginx:latest", spec.Container.Image)
	assert.NoError(t, spec.Validate())
}

func TestConverter_ConvertProject_MultipleServices(t *testing.T) {
	project := &types.Project{
		Name:       "multi",
		WorkingDir: "/test",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
			},
			"db": {
				Name:  "db",
				Image: "postgres:15",
			},
		},
	}

	converter := NewConverter("/test")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 2)

	names := []string{specs[0].Name, specs[1].Name}
	assert.ElementsMatch(t, []string{"multi-web", "multi-db"}, names)

	for _, spec := range specs {
		assert.NoError(t, spec.Validate())
	}
}

func TestConverter_ConvertProject_WithDependencies(t *testing.T) {
	project := &types.Project{
		Name:       "app",
		WorkingDir: "/test",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				DependsOn: map[string]types.ServiceDependency{
					"db":    {},
					"cache": {},
				},
			},
			"db": {
				Name:  "db",
				Image: "postgres:15",
			},
			"cache": {
				Name:  "cache",
				Image: "redis:7",
			},
		},
	}

	converter := NewConverter("/test")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)

	var webSpec *service.Spec
	for i := range specs {
		if specs[i].Name == "app-web" {
			webSpec = &specs[i]
			break
		}
	}
	require.NotNil(t, webSpec)

	// Dependencies should be sorted and prefixed
	assert.ElementsMatch(t, []string{"app-cache", "app-db"}, webSpec.DependsOn)
}

// ---------------------------
// Project validation
// ---------------------------

func TestConverter_ValidateProject_InvalidName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		wantErr     bool
	}{
		{
			name:        "valid simple name",
			projectName: "myapp",
			wantErr:     false,
		},
		{
			name:        "valid name with hyphen",
			projectName: "my-app",
			wantErr:     false,
		},
		{
			name:        "valid name with underscore",
			projectName: "my_app",
			wantErr:     false,
		},
		{
			name:        "valid name with dot",
			projectName: "my.app",
			wantErr:     false,
		},
		{
			name:        "invalid name starting with hyphen",
			projectName: "-myapp",
			wantErr:     true,
		},
		{
			name:        "invalid name with space",
			projectName: "my app",
			wantErr:     true,
		},
		{
			name:        "invalid name with special char",
			projectName: "my@app",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &types.Project{
				Name:       tt.projectName,
				WorkingDir: "/test",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			}

			converter := NewConverter("/test")
			_, err := converter.ConvertProject(project)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid project name")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConverter_ValidateProject_SwarmDriverRejected(t *testing.T) {
	tests := []struct {
		name    string
		project *types.Project
		wantErr string
	}{
		{
			name: "config with driver",
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
				Configs: map[string]types.ConfigObjConfig{
					"app-config": {
						Driver: "swarm",
					},
				},
				Services: types.Services{
					"web": {Name: "web", Image: "nginx:latest"},
				},
			},
			wantErr: "Swarm-specific",
		},
		{
			name: "secret with driver",
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
				Secrets: map[string]types.SecretConfig{
					"db-password": {
						Driver: "swarm",
					},
				},
				Services: types.Services{
					"web": {Name: "web", Image: "nginx:latest"},
				},
			},
			wantErr: "Swarm-specific",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewConverter("/test")
			_, err := converter.ConvertProject(tt.project)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// ---------------------------
// Sysctls conversion
// ---------------------------

func TestConverter_Sysctls(t *testing.T) {
	tests := []struct {
		name        string
		sysctls     map[string]string
		expected    map[string]string
		expectNil   bool
		expectEmpty bool
	}{
		{
			name: "single sysctl",
			sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
			},
			expected: map[string]string{
				"net.ipv4.ip_forward": "1",
			},
		},
		{
			name: "multiple sysctls",
			sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
				"net.core.somaxconn":  "1024",
			},
			expected: map[string]string{
				"net.ipv4.ip_forward": "1",
				"net.core.somaxconn":  "1024",
			},
		},
		{
			name: "kernel parameters",
			sysctls: map[string]string{
				"kernel.shmmax":                "68719476736",
				"kernel.shmall":                "4294967296",
				"net.ipv4.tcp_keepalive_time":  "600",
				"net.ipv4.tcp_keepalive_intvl": "60",
				"net.ipv4.conf.all.rp_filter":  "2",
			},
			expected: map[string]string{
				"kernel.shmmax":                "68719476736",
				"kernel.shmall":                "4294967296",
				"net.ipv4.tcp_keepalive_time":  "600",
				"net.ipv4.tcp_keepalive_intvl": "60",
				"net.ipv4.conf.all.rp_filter":  "2",
			},
		},
		{
			name:      "no sysctls",
			sysctls:   nil,
			expectNil: true,
		},
		{
			name:        "empty sysctls",
			sysctls:     map[string]string{},
			expected:    map[string]string{},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &types.Project{
				Name: "test",
				Services: types.Services{
					"app": {
						Name:    "app",
						Image:   "nginx:alpine",
						Sysctls: tt.sysctls,
					},
				},
			}

			converter := NewConverter("/tmp")
			specs, err := converter.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			if tt.expectNil {
				assert.Nil(t, specs[0].Container.Sysctls)
			} else if tt.expectEmpty {
				assert.NotNil(t, specs[0].Container.Sysctls)
				assert.Empty(t, specs[0].Container.Sysctls)
			} else {
				assert.Equal(t, tt.expected, specs[0].Container.Sysctls)
			}
		})
	}
}

// ---------------------------
// Namespace modes (pid/ipc/cgroup)
// ---------------------------

func TestConverter_NamespaceModes(t *testing.T) {
	tests := []struct {
		name       string
		pidMode    string
		ipcMode    string
		cgroupMode string
	}{
		{
			name:    "pid host",
			pidMode: "host",
		},
		{
			name:    "pid service reference",
			pidMode: "service:db",
		},
		{
			name:    "pid container reference",
			pidMode: "container:my-container",
		},
		{
			name:    "ipc host",
			ipcMode: "host",
		},
		{
			name:    "ipc shareable",
			ipcMode: "shareable",
		},
		{
			name:    "ipc container reference",
			ipcMode: "container:my-container",
		},
		{
			name:       "cgroup host",
			cgroupMode: "host",
		},
		{
			name:       "cgroup private",
			cgroupMode: "private",
		},
		{
			name:       "all namespace modes",
			pidMode:    "host",
			ipcMode:    "shareable",
			cgroupMode: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &types.Project{
				Name: "test",
				Services: types.Services{
					"web": {
						Name:   "web",
						Image:  "nginx:latest",
						Pid:    tt.pidMode,
						Ipc:    tt.ipcMode,
						Cgroup: tt.cgroupMode,
					},
				},
			}

			converter := NewConverter("/tmp")
			specs, err := converter.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			spec := specs[0]
			assert.Equal(t, tt.pidMode, spec.Container.PidMode)
			assert.Equal(t, tt.ipcMode, spec.Container.IpcMode)
			assert.Equal(t, tt.cgroupMode, spec.Container.CgroupMode)
		})
	}
}

// ---------------------------
// Network dependencies
// ---------------------------

func TestConverter_NetworkDependencies_ExplicitNetworks(t *testing.T) {
	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"frontend": {
				Name:   "frontend",
				Driver: "bridge",
			},
			"backend": {
				Name:   "backend",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"frontend": {},
					"backend":  {},
				},
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "myapp-web", spec.Name)

	// Service should have both networks in ServiceNetworks
	assert.ElementsMatch(t, []string{"myapp-backend", "myapp-frontend"}, spec.Container.Network.ServiceNetworks)

	// Spec.Networks should contain both networks
	require.Len(t, spec.Networks, 2)
	networkNames := []string{spec.Networks[0].Name, spec.Networks[1].Name}
	assert.ElementsMatch(t, []string{"myapp-backend", "myapp-frontend"}, networkNames)
}

func TestConverter_NetworkDependencies_ImplicitDefaultNetwork(t *testing.T) {
	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:     "web",
				Image:    "nginx:latest",
				Networks: nil, // No explicit networks
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Service without explicit networks should use project default networks
	assert.Contains(t, spec.Container.Network.ServiceNetworks, "myapp-default")
	require.Len(t, spec.Networks, 1)
	assert.Equal(t, "myapp-default", spec.Networks[0].Name)
}

func TestConverter_NetworkDependencies_MultipleDefaultNetworks(t *testing.T) {
	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
			"monitoring": {
				Name:   "monitoring",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:     "web",
				Image:    "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{}, // Empty but not nil
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Service with empty networks should get ALL project networks
	assert.ElementsMatch(t, []string{"myapp-default", "myapp-monitoring"}, spec.Container.Network.ServiceNetworks)
	require.Len(t, spec.Networks, 2)
	networkNames := []string{spec.Networks[0].Name, spec.Networks[1].Name}
	assert.ElementsMatch(t, []string{"myapp-default", "myapp-monitoring"}, networkNames)
}

func TestConverter_NetworkDependencies_ExternalNetwork(t *testing.T) {
	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
			"infrastructure-proxy": {
				Name:     "infrastructure-proxy",
				External: types.External(true),
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"default":              {},
					"infrastructure-proxy": {},
				},
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// External networks should be in ServiceNetworks
	assert.ElementsMatch(t, []string{"infrastructure-proxy", "myapp-default"}, spec.Container.Network.ServiceNetworks)

	// Spec.Networks should contain both networks
	require.Len(t, spec.Networks, 2)
	var externalNet *service.Network
	for i := range spec.Networks {
		if spec.Networks[i].Name == "infrastructure-proxy" {
			externalNet = &spec.Networks[i]
			break
		}
	}
	require.NotNil(t, externalNet)
	assert.True(t, externalNet.External)
}

func TestConverter_NetworkDependencies_ExternalNetworkNotInProject(t *testing.T) {
	project := &types.Project{
		Name: "llm",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"ollama": {
				Name:  "ollama",
				Image: "ollama:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"default":              {}, // Local network
					"infrastructure-proxy": {}, // External network NOT in project.Networks
				},
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// External network should be in ServiceNetworks WITHOUT project prefix
	assert.ElementsMatch(t,
		[]string{"infrastructure-proxy", "llm-default"},
		spec.Container.Network.ServiceNetworks)

	// Spec.Networks should have both networks
	require.Len(t, spec.Networks, 2)
	var externalNet *service.Network
	for i := range spec.Networks {
		if spec.Networks[i].Name == "infrastructure-proxy" {
			externalNet = &spec.Networks[i]
			break
		}
	}
	require.NotNil(t, externalNet)
	assert.True(t, externalNet.External)
}

func TestConverter_NetworkDependencies_BridgeMode(t *testing.T) {
	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"backend": {
				Name:   "backend",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"db": {
				Name:        "db",
				Image:       "postgres:15",
				NetworkMode: "bridge",
				Networks: map[string]*types.ServiceNetworkConfig{
					"backend": {},
				},
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Even with explicit bridge mode, service should have network in ServiceNetworks
	assert.Equal(t, "bridge", spec.Container.Network.Mode)
	assert.Contains(t, spec.Container.Network.ServiceNetworks, "myapp-backend")
}

func TestConverter_NetworkDependencies_NoNetworks(t *testing.T) {
	project := &types.Project{
		Name:     "myapp",
		Networks: map[string]types.NetworkConfig{},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
			},
		},
	}

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// When project has no networks, ServiceNetworks should be empty
	assert.Empty(t, spec.Container.Network.ServiceNetworks)
	assert.Empty(t, spec.Networks)
}

// ---------------------------
// Volume dependencies
// ---------------------------

func TestConverter_VolumeDependencies_ExplicitVolumes(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Service should only depend on volumes it actually uses
	assert.Len(t, spec.Volumes, 1)
	assert.Equal(t, "myapp-data", spec.Volumes[0].Name)
}

func TestConverter_VolumeDependencies_MultipleVolumes(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	assert.Len(t, spec.Volumes, 2)
	volumeNames := []string{spec.Volumes[0].Name, spec.Volumes[1].Name}
	assert.ElementsMatch(t, []string{"myapp-data", "myapp-logs"}, volumeNames)
}

func TestConverter_VolumeDependencies_NoVolumes(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Service without volume mounts should have no volume dependencies
	assert.Empty(t, spec.Volumes)
}

func TestConverter_VolumeDependencies_BindMountsOnly(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Bind mounts don't create volume unit dependencies
	assert.Empty(t, spec.Volumes)

	// But should still have the mounts in Container.Mounts
	assert.Len(t, spec.Container.Mounts, 2)
	assert.Equal(t, service.MountTypeBind, spec.Container.Mounts[0].Type)
	assert.Equal(t, service.MountTypeBind, spec.Container.Mounts[1].Type)
}

func TestConverter_VolumeDependencies_MixedMounts(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Should only depend on named volumes
	assert.Len(t, spec.Volumes, 1)
	assert.Equal(t, "myapp-data", spec.Volumes[0].Name)

	// All mounts should still be in Container.Mounts
	assert.Len(t, spec.Container.Mounts, 3)
}

func TestConverter_VolumeDependencies_ExternalVolumes(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Should have both volumes
	assert.Len(t, spec.Volumes, 2)

	var externalVol *service.Volume
	var localVol *service.Volume
	for i := range spec.Volumes {
		if spec.Volumes[i].External {
			externalVol = &spec.Volumes[i]
		} else {
			localVol = &spec.Volumes[i]
		}
	}

	require.NotNil(t, externalVol)
	require.NotNil(t, localVol)

	// External volume should NOT be prefixed
	assert.Equal(t, "shared-data", externalVol.Name)
	assert.True(t, externalVol.External)

	// Local volume should be prefixed
	assert.Equal(t, "myapp-local-data", localVol.Name)
	assert.False(t, localVol.External)
}

func TestConverter_VolumeDependencies_SharedVolume(t *testing.T) {
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

	converter := NewConverter(".")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 2)

	// Both services should have the shared volume
	for _, spec := range specs {
		assert.Len(t, spec.Volumes, 1)
		assert.Equal(t, "myapp-shared", spec.Volumes[0].Name)
	}
}

// ---------------------------
// Helper functions
// ---------------------------

func TestPrefix(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		resourceName string
		want         string
	}{
		{
			name:         "basic prefix",
			projectName:  "myapp",
			resourceName: "web",
			want:         "myapp-web",
		},
		{
			name:         "already prefixed with hyphen",
			projectName:  "myapp",
			resourceName: "myapp-web",
			want:         "myapp-web",
		},
		{
			name:         "already prefixed with underscore",
			projectName:  "myapp",
			resourceName: "myapp_web",
			want:         "myapp_web",
		},
		{
			name:         "partial match not considered prefixed",
			projectName:  "app",
			resourceName: "myapp-web",
			want:         "app-myapp-web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Prefix(tt.projectName, tt.resourceName)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFindEnvFiles(t *testing.T) {
	// Create temp directory with env files
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test env files
	envFiles := []string{
		".env",
		".env.web",
		"web.env",
	}
	for _, f := range envFiles {
		err := os.WriteFile(filepath.Join(tmpDir, f), []byte("TEST=1"), 0600)
		require.NoError(t, err)
	}

	// Create env subdirectory
	envDir := filepath.Join(tmpDir, "env")
	err = os.Mkdir(envDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(envDir, "web.env"), []byte("TEST=1"), 0600)
	require.NoError(t, err)

	found := FindEnvFiles("web", tmpDir)

	// Should find: .env, .env.web, web.env, env/web.env
	assert.Len(t, found, 4)
	assert.Contains(t, found, filepath.Join(tmpDir, ".env"))
	assert.Contains(t, found, filepath.Join(tmpDir, ".env.web"))
	assert.Contains(t, found, filepath.Join(tmpDir, "web.env"))
	assert.Contains(t, found, filepath.Join(tmpDir, "env", "web.env"))
}

func TestIsExternal(t *testing.T) {
	tests := []struct {
		name     string
		external interface{}
		want     bool
	}{
		{
			name:     "nil",
			external: nil,
			want:     false,
		},
		{
			name:     "bool true",
			external: true,
			want:     true,
		},
		{
			name:     "bool false",
			external: false,
			want:     false,
		},
		{
			name:     "types.External true",
			external: types.External(true),
			want:     true,
		},
		{
			name:     "types.External false",
			external: types.External(false),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExternal(tt.external)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------
// Resources conversion
// ---------------------------

func TestConverter_Resources(t *testing.T) {
	tests := []struct {
		name     string
		deploy   *types.DeployConfig
		service  types.ServiceConfig
		expected service.Resources
	}{
		{
			name: "memory limits",
			deploy: &types.DeployConfig{
				Resources: types.Resources{
					Limits: &types.Resource{
						MemoryBytes: types.UnitBytes(512 * 1024 * 1024),
					},
					Reservations: &types.Resource{
						MemoryBytes: types.UnitBytes(256 * 1024 * 1024),
					},
				},
			},
			expected: service.Resources{
				Memory:            "512m",
				MemoryReservation: "256m",
			},
		},
		{
			name: "cpu limits",
			deploy: &types.DeployConfig{
				Resources: types.Resources{
					Limits: &types.Resource{
						NanoCPUs: 1.5,
					},
				},
			},
			expected: service.Resources{
				CPUQuota:  150000,
				CPUPeriod: 100000,
			},
		},
		{
			name: "pids limit from deploy",
			deploy: &types.DeployConfig{
				Resources: types.Resources{
					Limits: &types.Resource{
						Pids: 100,
					},
				},
			},
			expected: service.Resources{
				PidsLimit: 100,
			},
		},
		{
			name: "shm size",
			service: types.ServiceConfig{
				ShmSize: types.UnitBytes(64 * 1024 * 1024),
			},
			expected: service.Resources{
				ShmSize: "64m",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := types.ServiceConfig{
				Name:    "app",
				Image:   "nginx:alpine",
				Deploy:  tt.deploy,
				ShmSize: tt.service.ShmSize,
			}

			project := &types.Project{
				Name:     "test",
				Services: types.Services{"app": svc},
			}

			converter := NewConverter("/tmp")
			specs, err := converter.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			assert.Equal(t, tt.expected.Memory, specs[0].Container.Resources.Memory)
			assert.Equal(t, tt.expected.MemoryReservation, specs[0].Container.Resources.MemoryReservation)
			assert.Equal(t, tt.expected.CPUQuota, specs[0].Container.Resources.CPUQuota)
			assert.Equal(t, tt.expected.CPUPeriod, specs[0].Container.Resources.CPUPeriod)
			assert.Equal(t, tt.expected.PidsLimit, specs[0].Container.Resources.PidsLimit)
			assert.Equal(t, tt.expected.ShmSize, specs[0].Container.Resources.ShmSize)
		})
	}
}

// ---------------------------
// Healthcheck conversion
// ---------------------------

func TestConverter_Healthcheck(t *testing.T) {
	interval := types.Duration(30 * time.Second)
	timeout := types.Duration(10 * time.Second)
	startPeriod := types.Duration(40 * time.Second)
	retries := uint64(3)

	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				HealthCheck: &types.HealthCheckConfig{
					Test:        []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
					Interval:    &interval,
					Timeout:     &timeout,
					StartPeriod: &startPeriod,
					Retries:     &retries,
				},
			},
		},
	}

	converter := NewConverter("/tmp")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	hc := specs[0].Container.Healthcheck
	require.NotNil(t, hc)
	assert.Equal(t, []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"}, hc.Test)
	assert.Equal(t, 30*time.Second, hc.Interval)
	assert.Equal(t, 10*time.Second, hc.Timeout)
	assert.Equal(t, 40*time.Second, hc.StartPeriod)
	assert.Equal(t, 3, hc.Retries)
}

func TestConverter_HealthcheckDisabled(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				HealthCheck: &types.HealthCheckConfig{
					Disable: true,
				},
			},
		},
	}

	converter := NewConverter("/tmp")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	assert.Nil(t, specs[0].Container.Healthcheck)
}
