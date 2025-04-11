package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestFromComposeVolume(t *testing.T) {
	// Test case 1: Volume with a name, driver, and driver options
	volumeName := "test-volume"
	composeVolume := types.VolumeConfig{
		Name:   "named-volume",
		Driver: "local",
		DriverOpts: map[string]string{
			"type":   "nfs",
			"device": "host:/path",
			"o":      "addr=192.168.1.1",
		},
		Labels: types.Labels{
			"com.example.description": "Test volume",
			"com.example.department":  "IT",
		},
	}

	volume := NewVolume(volumeName)
	volume = volume.FromComposeVolume(volumeName, composeVolume)

	// Verify the conversion was correct
	assert.Equal(t, volumeName, volume.Name)
	assert.Equal(t, "volume", volume.UnitType)

	// The volume name should be set to the specified name in the config
	assert.Equal(t, "named-volume", volume.VolumeName)

	// Driver should be set
	assert.Equal(t, "local", volume.Driver)

	// Driver options should be converted to options array
	assert.Contains(t, volume.Options, "type=nfs")
	assert.Contains(t, volume.Options, "device=host:/path")
	assert.Contains(t, volume.Options, "o=addr=192.168.1.1")

	// Labels should be transferred
	assert.Contains(t, volume.Label, "com.example.description=Test volume")
	assert.Contains(t, volume.Label, "com.example.department=IT")

	// Test case 2: Volume without a name (should use the key name)
	composeVolume2 := types.VolumeConfig{
		Driver: "local",
	}

	volume2Name := "data-volume"
	volume2 := NewVolume(volume2Name)
	volume2 = volume2.FromComposeVolume(volume2Name, composeVolume2)

	// The volume name should default to the key name when not specified
	assert.Equal(t, volume2Name, volume2.Name)
	assert.Equal(t, volume2Name, volume2.VolumeName)

	// Driver should be set
	assert.Equal(t, "local", volume2.Driver)
}