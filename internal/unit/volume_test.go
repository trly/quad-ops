package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestVolumeConfigYAMLMarshaling(t *testing.T) {
	// Create a sample volume config
	config := Volume{
		ContainersConfModule: []string{"module1", "module2"},
		Copy:                 true,
		Device:               "/dev/sda1",
		Driver:               "local",
		GlobalArgs:           []string{"--timeout=30s", "--debug"},
		Group:                "storage",
		Image:                "busybox:latest",
		Label:                []string{"environment=production", "app=database"},
		Options:              []string{"o=size=100G", "device=/dev/sda1"},
		PodmanArgs:           []string{"--log-level=debug", "--root=/var/lib/containers"},
		Type:                 "tmpfs",
		User:                 "1000",
		VolumeName:           "data-volume",
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Volume
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.ElementsMatch(t, config.ContainersConfModule, unmarshaled.ContainersConfModule)
	assert.Equal(t, config.Copy, unmarshaled.Copy)
	assert.Equal(t, config.Device, unmarshaled.Device)
	assert.Equal(t, config.Driver, unmarshaled.Driver)
	assert.ElementsMatch(t, config.GlobalArgs, unmarshaled.GlobalArgs)
	assert.Equal(t, config.Group, unmarshaled.Group)
	assert.Equal(t, config.Image, unmarshaled.Image)
	assert.ElementsMatch(t, config.Label, unmarshaled.Label)
	assert.ElementsMatch(t, config.Options, unmarshaled.Options)
	assert.ElementsMatch(t, config.PodmanArgs, unmarshaled.PodmanArgs)
	assert.Equal(t, config.Type, unmarshaled.Type)
	assert.Equal(t, config.User, unmarshaled.User)
	assert.Equal(t, config.VolumeName, unmarshaled.VolumeName)
}

func TestVolumeConfigYAMLUnmarshaling(t *testing.T) {
	yamlData := `
containers_conf_module:
  - volumeconf
  - storage
copy: false
device: /dev/nvme0n1
driver: btrfs
global_args:
  - --log-level=info
  - --storage-driver=btrfs
group: disk
image: alpine:latest
label:
  - purpose=backup
  - tier=storage
options:
  - size=500G
  - uid=1000
podman_args:
  - --runtime=runc
  - --events-backend=file
type: bind
user: root
volume_name: backup-storage
`

	var config Volume
	err := yaml.Unmarshal([]byte(yamlData), &config)
	assert.NoError(t, err)

	// Verify fields were properly unmarshaled
	assert.ElementsMatch(t, []string{"volumeconf", "storage"}, config.ContainersConfModule)
	assert.False(t, config.Copy)
	assert.Equal(t, "/dev/nvme0n1", config.Device)
	assert.Equal(t, "btrfs", config.Driver)
	assert.ElementsMatch(t, []string{"--log-level=info", "--storage-driver=btrfs"}, config.GlobalArgs)
	assert.Equal(t, "disk", config.Group)
	assert.Equal(t, "alpine:latest", config.Image)
	assert.ElementsMatch(t, []string{"purpose=backup", "tier=storage"}, config.Label)
	assert.ElementsMatch(t, []string{"size=500G", "uid=1000"}, config.Options)
	assert.ElementsMatch(t, []string{"--runtime=runc", "--events-backend=file"}, config.PodmanArgs)
	assert.Equal(t, "bind", config.Type)
	assert.Equal(t, "root", config.User)
	assert.Equal(t, "backup-storage", config.VolumeName)
}

func TestVolumeConfigYAMLMarshalingPartial(t *testing.T) {
	// Test with only some fields populated
	config := Volume{
		Driver:     "local",
		Type:       "volume",
		VolumeName: "simple-volume",
		// Other fields left empty
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Volume
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.Equal(t, "local", unmarshaled.Driver)
	assert.Equal(t, "volume", unmarshaled.Type)
	assert.Equal(t, "simple-volume", unmarshaled.VolumeName)
	assert.Empty(t, unmarshaled.ContainersConfModule)
	assert.False(t, unmarshaled.Copy)
	assert.Empty(t, unmarshaled.Device)
	assert.Empty(t, unmarshaled.GlobalArgs)
	assert.Empty(t, unmarshaled.Group)
	assert.Empty(t, unmarshaled.Image)
	assert.Empty(t, unmarshaled.Label)
	assert.Empty(t, unmarshaled.Options)
	assert.Empty(t, unmarshaled.PodmanArgs)
	assert.Empty(t, unmarshaled.User)
}

func TestVolumeConfigYAMLMarshalingEmpty(t *testing.T) {
	// Test with completely empty config
	config := Volume{}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Volume
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify all fields are empty/zero values
	assert.Empty(t, unmarshaled.ContainersConfModule)
	assert.False(t, unmarshaled.Copy)
	assert.Empty(t, unmarshaled.Device)
	assert.Empty(t, unmarshaled.Driver)
	assert.Empty(t, unmarshaled.GlobalArgs)
	assert.Empty(t, unmarshaled.Group)
	assert.Empty(t, unmarshaled.Image)
	assert.Empty(t, unmarshaled.Label)
	assert.Empty(t, unmarshaled.Options)
	assert.Empty(t, unmarshaled.PodmanArgs)
	assert.Empty(t, unmarshaled.Type)
	assert.Empty(t, unmarshaled.User)
	assert.Empty(t, unmarshaled.VolumeName)
}
