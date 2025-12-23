package systemd

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getVolValue is a helper to get a key value from the Volume section.
func getVolValue(unit Unit, key string) string {
	section := unit.File.Section("Volume")
	if section == nil {
		return ""
	}
	return section.Key(key).String()
}

// TestBuildVolume_BasicVolume tests that a simple volume creates the correct unit structure.
func TestBuildVolume_BasicVolume(t *testing.T) {
	vol := &types.VolumeConfig{}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "testproject-myvolume.volume", unit.Name)
	assert.NotNil(t, unit.File)
	assert.NotNil(t, unit.File.Section("Volume"))
}

// TestBuildVolume_WithDriver tests that the driver option is correctly mapped.
func TestBuildVolume_WithDriver(t *testing.T) {
	vol := &types.VolumeConfig{
		Driver: "local",
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "local", getVolValue(unit, "Driver"))
}

// TestBuildVolume_WithCustomName tests that a custom volume name is preserved.
func TestBuildVolume_WithCustomName(t *testing.T) {
	vol := &types.VolumeConfig{
		Name: "custom-volume-name",
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "custom-volume-name", getVolValue(unit, "VolumeName"))
}

// TestBuildVolume_WithLabels tests that labels are mapped with dot-notation.
func TestBuildVolume_WithLabels(t *testing.T) {
	vol := &types.VolumeConfig{
		Labels: types.Labels{
			"app": "myapp",
			"env": "production",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "myapp", getVolValue(unit, "Label.app"))
	assert.Equal(t, "production", getVolValue(unit, "Label.env"))
}

// TestBuildVolume_WithEmptyLabels tests that no Label keys are added when labels are empty.
func TestBuildVolume_WithEmptyLabels(t *testing.T) {
	vol := &types.VolumeConfig{
		Labels: types.Labels{},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	section := unit.File.Section("Volume")
	for _, key := range section.Keys() {
		assert.False(t, len(key.Name()) > 6 && key.Name()[:6] == "Label.")
	}
}

// TestBuildVolume_DriverOptsCopy tests the "copy" driver option mapping.
func TestBuildVolume_DriverOptsCopy(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"copy true", "true", true},
		{"copy false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vol := &types.VolumeConfig{
				DriverOpts: map[string]string{
					"copy": tt.value,
				},
			}
			unit := BuildVolume("testproject", "myvolume", vol)

			if tt.expected {
				assert.Equal(t, "true", getVolValue(unit, "Copy"))
			} else {
				assert.Empty(t, getVolValue(unit, "Copy"))
			}
		})
	}
}

// TestBuildVolume_DriverOptsDevice tests the "device" driver option mapping.
func TestBuildVolume_DriverOptsDevice(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"device": "tmpfs",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "tmpfs", getVolValue(unit, "Device"))
}

// TestBuildVolume_DriverOptsGroup tests the "group" driver option mapping.
func TestBuildVolume_DriverOptsGroup(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"group": "192",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "192", getVolValue(unit, "Group"))
}

// TestBuildVolume_DriverOptsImage tests the "image" driver option mapping.
func TestBuildVolume_DriverOptsImage(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"image": "quay.io/centos/centos:latest",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "quay.io/centos/centos:latest", getVolValue(unit, "Image"))
}

// TestBuildVolume_DriverOptsOptionsAlias tests the "options"/"o" driver option mapping.
func TestBuildVolume_DriverOptsOptionsAlias(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"options key", "options", "XYZ"},
		{"o key", "o", "ABC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vol := &types.VolumeConfig{
				DriverOpts: map[string]string{
					tt.key: tt.value,
				},
			}
			unit := BuildVolume("testproject", "myvolume", vol)
			assert.Equal(t, tt.value, getVolValue(unit, "Options"))
		})
	}
}

// TestBuildVolume_DriverOptsType tests the "type" driver option mapping.
func TestBuildVolume_DriverOptsType(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"type": "nfs",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "nfs", getVolValue(unit, "Type"))
}

// TestBuildVolume_DriverOptsUserAlias tests the "user"/"uid" driver option mapping.
func TestBuildVolume_DriverOptsUserAlias(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"user key", "user", "123"},
		{"uid key", "uid", "456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vol := &types.VolumeConfig{
				DriverOpts: map[string]string{
					tt.key: tt.value,
				},
			}
			unit := BuildVolume("testproject", "myvolume", vol)
			assert.Equal(t, tt.value, getVolValue(unit, "User"))
		})
	}
}

// TestBuildVolume_DriverOptsPath tests the legacy "path" driver option.
func TestBuildVolume_DriverOptsPath(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"path": "/mnt/data",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "/mnt/data", getVolValue(unit, "Path"))
}

// TestBuildVolume_DriverOptsPathEmpty tests that empty path is not added.
func TestBuildVolume_DriverOptsPathEmpty(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"path": "",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Empty(t, getVolValue(unit, "Path"))
}

// TestBuildVolume_DriverOptsContainersConfModule tests the containers-conf-module driver option.
func TestBuildVolume_DriverOptsContainersConfModule(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"containers-conf-module": "mymodule",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "mymodule", getVolValue(unit, "ContainersConfModule"))
}

// TestBuildVolume_DriverOptsModule tests the "module" driver option (alias for containers-conf-module).
func TestBuildVolume_DriverOptsModule(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"module": "testmodule",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "testmodule", getVolValue(unit, "ContainersConfModule"))
}

// TestBuildVolume_DriverOptsUnknown tests that unknown driver options are ignored.
func TestBuildVolume_DriverOptsUnknown(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"unknown-option":  "value",
			"another-unknown": "data",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Empty(t, getVolValue(unit, "unknown-option"))
	assert.Empty(t, getVolValue(unit, "another-unknown"))
}

// TestBuildVolume_MultipleDriverOpts tests multiple driver options together.
func TestBuildVolume_MultipleDriverOpts(t *testing.T) {
	vol := &types.VolumeConfig{
		DriverOpts: map[string]string{
			"device":  "tmpfs",
			"type":    "tmpfs",
			"options": "nodev,noexec",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "tmpfs", getVolValue(unit, "Device"))
	assert.Equal(t, "tmpfs", getVolValue(unit, "Type"))
	assert.Equal(t, "nodev,noexec", getVolValue(unit, "Options"))
}

// TestBuildVolume_ExtensionGlobalArgs tests x-quad-ops-podman-args extension parsing.
func TestBuildVolume_ExtensionGlobalArgs(t *testing.T) {
	vol := &types.VolumeConfig{
		Extensions: map[string]interface{}{
			"x-quad-ops-podman-args": []interface{}{
				"--log-driver=json-file",
				"--log-opt=max-size=10m",
			},
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "--log-driver=json-file", getVolValue(unit, "GlobalArgs.0"))
	assert.Equal(t, "--log-opt=max-size=10m", getVolValue(unit, "GlobalArgs.1"))
}

// TestBuildVolume_ExtensionVolumeArgs tests x-quad-ops-volume-args extension parsing.
func TestBuildVolume_ExtensionVolumeArgs(t *testing.T) {
	vol := &types.VolumeConfig{
		Extensions: map[string]interface{}{
			"x-quad-ops-volume-args": []interface{}{
				"--opt=custom-opt",
				"--label=vol-specific",
			},
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "--opt=custom-opt", getVolValue(unit, "PodmanArgs.0"))
	assert.Equal(t, "--label=vol-specific", getVolValue(unit, "PodmanArgs.1"))
}

// TestBuildVolume_ExtensionGlobalArgsInvalid tests that non-string items in global args are skipped.
func TestBuildVolume_ExtensionGlobalArgsInvalid(t *testing.T) {
	vol := &types.VolumeConfig{
		Extensions: map[string]interface{}{
			"x-quad-ops-podman-args": []interface{}{
				"--log-driver=json-file",
				123, // non-string, should be skipped
				"--log-opt=max-size=10m",
			},
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "--log-driver=json-file", getVolValue(unit, "GlobalArgs.0"))
	// Non-string at index 1 means GlobalArgs.1 is not set
	assert.Equal(t, "--log-opt=max-size=10m", getVolValue(unit, "GlobalArgs.2"))
}

// TestBuildVolume_ExtensionVolumeArgsInvalid tests that non-string items in volume args are skipped.
func TestBuildVolume_ExtensionVolumeArgsInvalid(t *testing.T) {
	vol := &types.VolumeConfig{
		Extensions: map[string]interface{}{
			"x-quad-ops-volume-args": []interface{}{
				"--opt=first",
				map[string]interface{}{}, // non-string, should be skipped
				"--opt=third",
			},
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "--opt=first", getVolValue(unit, "PodmanArgs.0"))
	// Non-string at index 1 means PodmanArgs.1 is not set
	assert.Equal(t, "--opt=third", getVolValue(unit, "PodmanArgs.2"))
}

// TestBuildVolume_EmptyExtensions tests that missing extensions are handled gracefully.
func TestBuildVolume_EmptyExtensions(t *testing.T) {
	vol := &types.VolumeConfig{
		Extensions: map[string]interface{}{},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	section := unit.File.Section("Volume")
	for _, key := range section.Keys() {
		assert.NotContains(t, key.Name(), "GlobalArgs")
		assert.NotContains(t, key.Name(), "PodmanArgs")
	}
}

// TestBuildVolume_AllFieldsTogether tests a volume with all fields set together.
func TestBuildVolume_AllFieldsTogether(t *testing.T) {
	vol := &types.VolumeConfig{
		Driver: "local",
		Name:   "full-volume",
		Labels: types.Labels{
			"owner": "admin",
		},
		DriverOpts: map[string]string{
			"device":  "/dev/sda1",
			"type":    "ext4",
			"options": "rw,relatime",
			"user":    "1000",
		},
		Extensions: map[string]interface{}{
			"x-quad-ops-podman-args": []interface{}{
				"--log-driver=journald",
			},
			"x-quad-ops-volume-args": []interface{}{
				"--label=managed=true",
			},
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "local", getVolValue(unit, "Driver"))
	assert.Equal(t, "full-volume", getVolValue(unit, "VolumeName"))
	assert.Equal(t, "admin", getVolValue(unit, "Label.owner"))
	assert.Equal(t, "/dev/sda1", getVolValue(unit, "Device"))
	assert.Equal(t, "ext4", getVolValue(unit, "Type"))
	assert.Equal(t, "rw,relatime", getVolValue(unit, "Options"))
	assert.Equal(t, "1000", getVolValue(unit, "User"))
	assert.Equal(t, "--log-driver=journald", getVolValue(unit, "GlobalArgs.0"))
	assert.Equal(t, "--label=managed=true", getVolValue(unit, "PodmanArgs.0"))
}

// TestBuildVolume_MultipleLabels tests that multiple labels are all preserved.
func TestBuildVolume_MultipleLabels(t *testing.T) {
	vol := &types.VolumeConfig{
		Labels: types.Labels{
			"app":       "database",
			"component": "persistent-store",
			"version":   "v2",
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	assert.Equal(t, "database", getVolValue(unit, "Label.app"))
	assert.Equal(t, "persistent-store", getVolValue(unit, "Label.component"))
	assert.Equal(t, "v2", getVolValue(unit, "Label.version"))
}

// TestBuildVolume_NameDerivation tests that the unit name is derived from the project and volume name.
func TestBuildVolume_NameDerivation(t *testing.T) {
	tests := []struct {
		project      string
		volume       string
		expectedUnit string
	}{
		{"myproject", "data", "myproject-data.volume"},
		{"myproject", "db-storage", "myproject-db-storage.volume"},
		{"myproject", "cache_vol", "myproject-cache_vol.volume"},
	}

	for _, tt := range tests {
		t.Run(tt.volume, func(t *testing.T) {
			vol := &types.VolumeConfig{}
			unit := BuildVolume(tt.project, tt.volume, vol)
			assert.Equal(t, tt.expectedUnit, unit.Name)
		})
	}
}

// TestBuildVolume_SectionStructure tests that the unit always has a Volume section.
func TestBuildVolume_SectionStructure(t *testing.T) {
	vol := &types.VolumeConfig{}
	unit := BuildVolume("testproject", "vol", vol)

	require.NotNil(t, unit.File)
	require.NotNil(t, unit.File.Section("Volume"))
}

// TestBuildVolumeSection_NoDriver tests that empty driver is not added.
func TestBuildVolumeSection_NoDriver(t *testing.T) {
	vol := &types.VolumeConfig{
		Driver: "",
	}
	unit := BuildVolume("testproject", "vol", vol)

	assert.Empty(t, getVolValue(unit, "Driver"))
}

// TestBuildVolumeSection_NoVolumeName tests that missing Name field doesn't add VolumeName.
func TestBuildVolumeSection_NoVolumeName(t *testing.T) {
	vol := &types.VolumeConfig{
		Name: "",
	}
	unit := BuildVolume("testproject", "vol", vol)

	assert.Empty(t, getVolValue(unit, "VolumeName"))
}

// TestBuildVolume_ExtensionWrongType tests that non-slice extensions are ignored.
func TestBuildVolume_ExtensionWrongType(t *testing.T) {
	vol := &types.VolumeConfig{
		Extensions: map[string]interface{}{
			"x-quad-ops-podman-args": "not-a-slice",
			"x-quad-ops-volume-args": map[string]string{},
		},
	}
	unit := BuildVolume("testproject", "myvolume", vol)

	section := unit.File.Section("Volume")
	for _, key := range section.Keys() {
		assert.NotContains(t, key.Name(), "GlobalArgs")
		assert.NotContains(t, key.Name(), "PodmanArgs")
	}
}
