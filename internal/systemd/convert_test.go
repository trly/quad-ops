package systemd

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert_SkipsExternalNetworks(t *testing.T) {
	project := &types.Project{
		Name: "testproject",
		Networks: types.Networks{
			"internal-net": types.NetworkConfig{
				Driver: "bridge",
			},
			"external-net": types.NetworkConfig{
				Name:     "infrastructure-proxy",
				External: true,
			},
		},
	}

	units, err := Convert(project)
	require.NoError(t, err)

	var networkUnits []Unit
	for _, u := range units {
		if hasExtension(u.Name, ".network") {
			networkUnits = append(networkUnits, u)
		}
	}

	assert.Len(t, networkUnits, 1, "should only generate one network unit for non-external network")
	assert.Equal(t, "testproject-internal-net.network", networkUnits[0].Name)
}

func TestConvert_SkipsExternalVolumes(t *testing.T) {
	project := &types.Project{
		Name: "testproject",
		Volumes: types.Volumes{
			"internal-vol": types.VolumeConfig{
				Driver: "local",
			},
			"external-vol": types.VolumeConfig{
				Name:     "shared-data",
				External: true,
			},
		},
	}

	units, err := Convert(project)
	require.NoError(t, err)

	var volumeUnits []Unit
	for _, u := range units {
		if hasExtension(u.Name, ".volume") {
			volumeUnits = append(volumeUnits, u)
		}
	}

	assert.Len(t, volumeUnits, 1, "should only generate one volume unit for non-external volume")
	assert.Equal(t, "testproject-internal-vol.volume", volumeUnits[0].Name)
}

func TestConvert_AllExternalNetworksProducesNoNetworkUnits(t *testing.T) {
	project := &types.Project{
		Name: "testproject",
		Networks: types.Networks{
			"proxy": types.NetworkConfig{
				Name:     "infrastructure-proxy",
				External: true,
			},
		},
	}

	units, err := Convert(project)
	require.NoError(t, err)

	for _, u := range units {
		assert.False(t, hasExtension(u.Name, ".network"),
			"no .network units should be generated when all networks are external")
	}
}

func TestConvert_ResolvesRelativeBindMountPaths(t *testing.T) {
	project := &types.Project{
		Name:       "testproject",
		WorkingDir: "/srv/repos/myapp",
		Services: types.Services{
			"web": types.ServiceConfig{
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   types.VolumeTypeBind,
						Source: "./Caddyfile",
						Target: "/Caddyfile",
					},
					{
						Type:   types.VolumeTypeBind,
						Source: "config/nginx.conf",
						Target: "/etc/nginx/nginx.conf",
					},
					{
						Type:   types.VolumeTypeBind,
						Source: "/etc/ssl/certs",
						Target: "/certs",
					},
					{
						Type:   types.VolumeTypeVolume,
						Source: "data",
						Target: "/data",
					},
				},
			},
		},
	}

	units, err := Convert(project)
	require.NoError(t, err)

	var containerUnit Unit
	for _, u := range units {
		if hasExtension(u.Name, ".container") {
			containerUnit = u
		}
	}

	vals := containerUnit.File.Section("Container").Key("Volume").ValueWithShadows()
	require.Len(t, vals, 4)

	// Relative paths should be resolved to absolute
	assert.Contains(t, vals[0]+vals[1]+vals[2]+vals[3], "/srv/repos/myapp/Caddyfile:")
	assert.Contains(t, vals[0]+vals[1]+vals[2]+vals[3], "/srv/repos/myapp/config/nginx.conf:")

	// Absolute bind mount should remain unchanged
	assert.Contains(t, vals[0]+vals[1]+vals[2]+vals[3], "/etc/ssl/certs:/certs:")

	// Named volume should reference the Quadlet .volume unit
	assert.Contains(t, vals[0]+vals[1]+vals[2]+vals[3], "testproject-data.volume:/data:")
}

func hasExtension(name, ext string) bool {
	return len(name) > len(ext) && name[len(name)-len(ext):] == ext
}
