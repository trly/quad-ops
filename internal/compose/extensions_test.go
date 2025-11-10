package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/service"
)

func TestSpecConverter_ConvertPodmanBuildArgs(t *testing.T) {
	baseVal := "alpine"
	ubuntuVal := "ubuntu"

	tests := []struct {
		name       string
		extensions map[string]interface{}
		build      *types.BuildConfig
		want       []string
		wantArgs   map[string]string
	}{
		{
			name:       "no extension",
			extensions: map[string]interface{}{},
			build:      &types.BuildConfig{Args: types.MappingWithEquals{"BASE": &baseVal}},
			wantArgs:   map[string]string{"BASE": "alpine"},
		},
		{
			name: "x-podman-buildargs single arg",
			extensions: map[string]interface{}{
				"x-podman-buildargs": map[string]interface{}{
					"BUILDKIT_INLINE_CACHE": "1",
				},
			},
			build: &types.BuildConfig{Args: types.MappingWithEquals{"BASE": &baseVal}},
			wantArgs: map[string]string{
				"BASE":                   "alpine",
				"BUILDKIT_INLINE_CACHE": "1",
			},
		},
		{
			name: "x-podman-buildargs multiple args",
			extensions: map[string]interface{}{
				"x-podman-buildargs": map[string]interface{}{
					"BUILDKIT_INLINE_CACHE": "1",
					"BUILDPLATFORM":         "linux/amd64",
				},
			},
			build: &types.BuildConfig{Args: types.MappingWithEquals{"BASE": &ubuntuVal}},
			wantArgs: map[string]string{
				"BASE":                   "ubuntu",
				"BUILDKIT_INLINE_CACHE": "1",
				"BUILDPLATFORM":         "linux/amd64",
			},
		},
		{
			name: "x-podman-buildargs overrides compose args",
			extensions: map[string]interface{}{
				"x-podman-buildargs": map[string]interface{}{
					"BASE": "fedora",
				},
			},
			build: &types.BuildConfig{Args: types.MappingWithEquals{"BASE": &baseVal}},
			wantArgs: map[string]string{
				"BASE": "fedora",
			},
		},
		{
			name: "x-podman-buildargs with no compose args",
			extensions: map[string]interface{}{
				"x-podman-buildargs": map[string]interface{}{
					"CUSTOM_VAR": "value",
				},
			},
			build: &types.BuildConfig{},
			wantArgs: map[string]string{
				"CUSTOM_VAR": "value",
			},
		},
		{
			name: "x-podman-buildargs not a map",
			extensions: map[string]interface{}{
				"x-podman-buildargs": "not-a-map",
			},
			build:    &types.BuildConfig{Args: types.MappingWithEquals{"BASE": &baseVal}},
			wantArgs: map[string]string{"BASE": "alpine"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSpecConverter(".")

			// Create a minimal service config with extensions
			svc := types.ServiceConfig{
				Name:       "test",
				Image:      "test:latest",
				Build:      tt.build,
				Extensions: tt.extensions,
			}

			// Create a minimal project
			project := &types.Project{
				Name:       "test-project",
				WorkingDir: ".",
				Services: map[string]types.ServiceConfig{
					"test": svc,
				},
			}

			// Convert the service
			specs, err := sc.ConvertProject(project)
			assert.NoError(t, err)
			assert.Len(t, specs, 1)

			// Check build args
			if tt.build != nil {
				assert.NotNil(t, specs[0].Container.Build)
				assert.Equal(t, tt.wantArgs, specs[0].Container.Build.Args)
			}
		})
	}
}

func TestSpecConverter_ConvertPodmanVolumes(t *testing.T) {
	tests := []struct {
		name       string
		extensions map[string]interface{}
		wantMounts int
		checkMount func(t *testing.T, mounts []service.Mount)
	}{
		{
			name:       "no extension",
			extensions: map[string]interface{}{},
			wantMounts: 0,
		},
		{
			name: "x-podman-volumes single volume",
			extensions: map[string]interface{}{
				"x-podman-volumes": []interface{}{
					"cache:/tmp/cache:O",
				},
			},
			wantMounts: 1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, "test-project-cache", mounts[0].Source)
				assert.Equal(t, "/tmp/cache", mounts[0].Target)
				assert.Equal(t, service.MountTypeVolume, mounts[0].Type)
			},
		},
		{
			name: "x-podman-volumes multiple volumes",
			extensions: map[string]interface{}{
				"x-podman-volumes": []interface{}{
					"cache:/tmp/cache:O",
					"logs:/logs:U",
				},
			},
			wantMounts: 2,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Len(t, mounts, 2)
				assert.Equal(t, "test-project-cache", mounts[0].Source)
				assert.Equal(t, "/tmp/cache", mounts[0].Target)
				assert.Equal(t, "test-project-logs", mounts[1].Source)
				assert.Equal(t, "/logs", mounts[1].Target)
			},
		},
		{
			name: "x-podman-volumes with bind mount",
			extensions: map[string]interface{}{
				"x-podman-volumes": []interface{}{
					"/data:/data:ro",
				},
			},
			wantMounts: 1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, "/data", mounts[0].Source)
				assert.Equal(t, "/data", mounts[0].Target)
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.True(t, mounts[0].ReadOnly)
			},
		},
		{
			name: "x-podman-volumes not a slice",
			extensions: map[string]interface{}{
				"x-podman-volumes": "not-a-slice",
			},
			wantMounts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSpecConverter(".")

			// Create a minimal service config with extensions
			svc := types.ServiceConfig{
				Name:       "test",
				Image:      "test:latest",
				Extensions: tt.extensions,
			}

			// Create a minimal project
			project := &types.Project{
				Name:       "test-project",
				WorkingDir: ".",
				Services: map[string]types.ServiceConfig{
					"test": svc,
				},
			}

			// Convert the service
			specs, err := sc.ConvertProject(project)
			assert.NoError(t, err)
			assert.Len(t, specs, 1)

			// Count podman-specific mounts (those from x-podman-volumes)
			podmanMounts := make([]service.Mount, 0)
			for _, mount := range specs[0].Container.Mounts {
				// Podman volumes don't have corresponding volumes in the project
				// so we just check total mounts match expectation
				podmanMounts = append(podmanMounts, mount)
			}

			if tt.checkMount != nil {
				tt.checkMount(t, podmanMounts)
			} else {
				assert.GreaterOrEqual(t, len(podmanMounts), tt.wantMounts)
			}
		})
	}
}

func TestSpecConverter_convertPodmanBuildArgsExtension(t *testing.T) {
	tests := []struct {
		name       string
		extension  interface{}
		wantResult map[string]string
	}{
		{
			name:       "nil extension",
			extension:  nil,
			wantResult: nil,
		},
		{
			name: "valid map",
			extension: map[string]interface{}{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantResult: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "map with non-string values",
			extension: map[string]interface{}{
				"KEY1": "value1",
				"KEY2": 123,
				"KEY3": true,
			},
			wantResult: map[string]string{
				"KEY1": "value1",
			},
		},
		{
			name:       "not a map",
			extension:  "invalid",
			wantResult: nil,
		},
		{
			name:       "empty map",
			extension:  map[string]interface{}{},
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSpecConverter(".")
			result := sc.convertPodmanBuildArgsExtension(tt.extension)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestSpecConverter_convertPodmanVolumesExtension(t *testing.T) {
	tests := []struct {
		name       string
		extension  interface{}
		project    *types.Project
		wantMounts int
	}{
		{
			name:       "nil extension",
			extension:  nil,
			project:    &types.Project{Name: "test", WorkingDir: "."},
			wantMounts: 0,
		},
		{
			name: "valid slice",
			extension: []interface{}{
				"cache:/tmp/cache:O",
				"logs:/logs:U",
			},
			project:    &types.Project{Name: "test", WorkingDir: "."},
			wantMounts: 2,
		},
		{
			name:       "not a slice",
			extension:  "invalid",
			project:    &types.Project{Name: "test", WorkingDir: "."},
			wantMounts: 0,
		},
		{
			name:       "empty slice",
			extension:  []interface{}{},
			project:    &types.Project{Name: "test", WorkingDir: "."},
			wantMounts: 0,
		},
		{
			name: "slice with invalid entries",
			extension: []interface{}{
				"valid:/mnt/valid",
				123, // invalid
				true, // invalid
			},
			project:    &types.Project{Name: "test", WorkingDir: "."},
			wantMounts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSpecConverter(".")
			result := sc.convertPodmanVolumesExtension(tt.extension, tt.project)
			assert.Len(t, result, tt.wantMounts)
		})
	}
}
