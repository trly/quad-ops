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

func hasExtension(name, ext string) bool {
	return len(name) > len(ext) && name[len(name)-len(ext):] == ext
}
