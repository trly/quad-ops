package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestImageConfigYAMLMarshaling(t *testing.T) {
	// Create a sample image config
	config := Image{
		Image:      "alpine:latest",
		PodmanArgs: []string{"--platform=linux/amd64", "--pull=always"},
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Image
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.Equal(t, config.Image, unmarshaled.Image)
	assert.ElementsMatch(t, config.PodmanArgs, unmarshaled.PodmanArgs)
}

func TestImageConfigYAMLUnmarshaling(t *testing.T) {
	yamlData := `
image: fedora:36
podman_args:
  - --log-level=debug
  - --cgroup-manager=systemd
  - --runtime=crun
`

	var config Image
	err := yaml.Unmarshal([]byte(yamlData), &config)
	assert.NoError(t, err)

	// Verify fields were properly unmarshaled
	assert.Equal(t, "fedora:36", config.Image)
	assert.ElementsMatch(t, []string{"--log-level=debug", "--cgroup-manager=systemd", "--runtime=crun"}, config.PodmanArgs)
}

func TestImageConfigYAMLMarshalingEmpty(t *testing.T) {
	// Test with empty fields
	config := Image{
		Image: "busybox:latest",
		// Empty PodmanArgs
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Image
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.Equal(t, config.Image, unmarshaled.Image)
	assert.Empty(t, unmarshaled.PodmanArgs)
}
