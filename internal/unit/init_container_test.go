package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestParseInitContainers(t *testing.T) {
	tests := []struct {
		name      string
		service   types.ServiceConfig
		expected  []InitContainer
		expectErr bool
	}{
		{
			name: "No init containers",
			service: types.ServiceConfig{
				Name: "test-service",
			},
			expected:  nil,
			expectErr: false,
		},
		{
			name: "Single init container with string command",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": []interface{}{
						map[string]interface{}{
							"image":   "docker.io/example/image:tag",
							"command": "run this command",
						},
					},
				},
			},
			expected: []InitContainer{
				{
					Image:   "docker.io/example/image:tag",
					Command: []string{"run this command"},
				},
			},
			expectErr: false,
		},
		{
			name: "Single init container with array command",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": []interface{}{
						map[string]interface{}{
							"image":   "alpine:latest",
							"command": []interface{}{"sh", "-c", "echo hello"},
						},
					},
				},
			},
			expected: []InitContainer{
				{
					Image:   "alpine:latest",
					Command: []string{"sh", "-c", "echo hello"},
				},
			},
			expectErr: false,
		},
		{
			name: "Multiple init containers",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": []interface{}{
						map[string]interface{}{
							"image":   "alpine:latest",
							"command": "echo first",
						},
						map[string]interface{}{
							"image":   "busybox:latest",
							"command": "echo second",
						},
					},
				},
			},
			expected: []InitContainer{
				{
					Image:   "alpine:latest",
					Command: []string{"echo first"},
				},
				{
					Image:   "busybox:latest",
					Command: []string{"echo second"},
				},
			},
			expectErr: false,
		},
		{
			name: "Init container without command",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": []interface{}{
						map[string]interface{}{
							"image": "alpine:latest",
						},
					},
				},
			},
			expected: []InitContainer{
				{
					Image:   "alpine:latest",
					Command: nil,
				},
			},
			expectErr: false,
		},
		{
			name: "Invalid extension format - not a list",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": "invalid",
				},
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "Invalid init container - not a map",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": []interface{}{
						"invalid",
					},
				},
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "Missing image field",
			service: types.ServiceConfig{
				Name: "test-service",
				Extensions: map[string]interface{}{
					"x-quad-ops-init": []interface{}{
						map[string]interface{}{
							"command": "echo hello",
						},
					},
				},
			},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseInitContainers(tt.service)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCreateInitContainerUnit(t *testing.T) {
	tests := []struct {
		name          string
		initContainer InitContainer
		initName      string
		expected      *Container
	}{
		{
			name: "Init container with command",
			initContainer: InitContainer{
				Image:   "alpine:latest",
				Command: []string{"sh", "-c", "echo hello"},
			},
			initName: "test-project-test-service-init-0",
			expected: &Container{
				BaseUnit: BaseUnit{
					BaseUnit: nil, // This will be set by NewContainer
					Name:     "test-project-test-service-init-0",
				},
				Image: "alpine:latest",
				Exec:  []string{"sh", "-c", "echo hello"},
			},
		},
		{
			name: "Init container without command",
			initContainer: InitContainer{
				Image: "busybox:latest",
			},
			initName: "test-project-test-service-init-1",
			expected: &Container{
				BaseUnit: BaseUnit{
					BaseUnit: nil, // This will be set by NewContainer
					Name:     "test-project-test-service-init-1",
				},
				Image: "busybox:latest",
				Exec:  nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal parent container for testing
			parentContainer := NewContainer("parent")
			result := CreateInitContainerUnit(tt.initContainer, tt.initName, parentContainer)

			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Image, result.Image)
			assert.Equal(t, tt.expected.Exec, result.Exec)
			assert.NotNil(t, result.BaseUnit.BaseUnit)
		})
	}
}
