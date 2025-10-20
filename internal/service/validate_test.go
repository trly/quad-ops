package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "Name",
		Message: "is required",
	}
	assert.Equal(t, "Name: is required", err.Error())
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   ValidationErrors
		expected string
	}{
		{
			name:     "empty errors",
			errors:   ValidationErrors{},
			expected: "",
		},
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "Name", Message: "is required"},
			},
			expected: "Name: is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "Name", Message: "is required"},
				{Field: "Image", Message: "is invalid"},
			},
			expected: "Name: is required; Image: is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.errors.Error())
		})
	}
}

func TestSpec_Validate(t *testing.T) {
	tests := []struct {
		name        string
		spec        Spec
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid spec",
			spec: Spec{
				Name: "test-service",
				Container: Container{
					Image: "nginx:latest",
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			spec: Spec{
				Container: Container{
					Image: "nginx:latest",
				},
			},
			expectError: true,
			errorMsg:    "service name is required",
		},
		{
			name: "invalid name - starts with dash",
			spec: Spec{
				Name: "-invalid",
				Container: Container{
					Image: "nginx:latest",
				},
			},
			expectError: true,
			errorMsg:    "invalid service name",
		},
		{
			name: "invalid name - contains spaces",
			spec: Spec{
				Name: "invalid name",
				Container: Container{
					Image: "nginx:latest",
				},
			},
			expectError: true,
			errorMsg:    "invalid service name",
		},
		{
			name: "invalid name - special characters",
			spec: Spec{
				Name: "test@service",
				Container: Container{
					Image: "nginx:latest",
				},
			},
			expectError: true,
			errorMsg:    "invalid service name",
		},
		{
			name: "valid name with dots and hyphens",
			spec: Spec{
				Name: "my-service.v1",
				Container: Container{
					Image: "nginx:latest",
				},
			},
			expectError: false,
		},
		{
			name: "self-dependency",
			spec: Spec{
				Name: "web",
				Container: Container{
					Image: "nginx:latest",
				},
				DependsOn: []string{"web"},
			},
			expectError: true,
			errorMsg:    "service cannot depend on itself",
		},
		{
			name: "invalid container",
			spec: Spec{
				Name:      "test",
				Container: Container{}, // No image
			},
			expectError: true,
			errorMsg:    "image is required",
		},
		{
			name: "invalid volume",
			spec: Spec{
				Name: "test",
				Container: Container{
					Image: "nginx:latest",
				},
				Volumes: []Volume{
					{Name: ""}, // Empty name
				},
			},
			expectError: true,
			errorMsg:    "volume name is required",
		},
		{
			name: "invalid network",
			spec: Spec{
				Name: "test",
				Container: Container{
					Image: "nginx:latest",
				},
				Networks: []Network{
					{Name: ""}, // Empty name
				},
			},
			expectError: true,
			errorMsg:    "network name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContainer_Validate(t *testing.T) {
	tests := []struct {
		name        string
		container   Container
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid with image",
			container: Container{
				Image: "nginx:latest",
			},
			expectError: false,
		},
		{
			name: "valid with build",
			container: Container{
				Build: &Build{
					Context: ".",
				},
			},
			expectError: false,
		},
		{
			name:        "missing image and build",
			container:   Container{},
			expectError: true,
			errorMsg:    "image is required when build is not specified",
		},
		{
			name: "invalid restart policy",
			container: Container{
				Image:         "nginx:latest",
				RestartPolicy: "invalid-policy",
			},
			expectError: true,
			errorMsg:    "invalid restart policy",
		},
		{
			name: "valid restart policy - no",
			container: Container{
				Image:         "nginx:latest",
				RestartPolicy: RestartPolicyNo,
			},
			expectError: false,
		},
		{
			name: "valid restart policy - always",
			container: Container{
				Image:         "nginx:latest",
				RestartPolicy: RestartPolicyAlways,
			},
			expectError: false,
		},
		{
			name: "valid restart policy - on-failure",
			container: Container{
				Image:         "nginx:latest",
				RestartPolicy: RestartPolicyOnFailure,
			},
			expectError: false,
		},
		{
			name: "valid restart policy - unless-stopped",
			container: Container{
				Image:         "nginx:latest",
				RestartPolicy: RestartPolicyUnlessStopped,
			},
			expectError: false,
		},
		{
			name: "invalid healthcheck",
			container: Container{
				Image: "nginx:latest",
				Healthcheck: &Healthcheck{
					Test: []string{}, // Empty test
				},
			},
			expectError: true,
			errorMsg:    "healthcheck test command is required",
		},
		{
			name: "invalid build",
			container: Container{
				Build: &Build{
					Context: "", // Empty context
				},
			},
			expectError: true,
			errorMsg:    "build context is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.container.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHealthcheck_Validate(t *testing.T) {
	tests := []struct {
		name        string
		healthcheck Healthcheck
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid healthcheck",
			healthcheck: Healthcheck{
				Test:     []string{"CMD", "curl", "-f", "http://localhost"},
				Interval: 30 * time.Second,
				Timeout:  3 * time.Second,
				Retries:  3,
			},
			expectError: false,
		},
		{
			name: "missing test",
			healthcheck: Healthcheck{
				Interval: 30 * time.Second,
			},
			expectError: true,
			errorMsg:    "healthcheck test command is required",
		},
		{
			name: "negative retries",
			healthcheck: Healthcheck{
				Test:    []string{"CMD", "test"},
				Retries: -1,
			},
			expectError: true,
			errorMsg:    "retries must be non-negative",
		},
		{
			name: "zero retries is valid",
			healthcheck: Healthcheck{
				Test:    []string{"CMD", "test"},
				Retries: 0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.healthcheck.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuild_Validate(t *testing.T) {
	tests := []struct {
		name        string
		build       Build
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid build",
			build: Build{
				Context: ".",
			},
			expectError: false,
		},
		{
			name: "valid with dockerfile",
			build: Build{
				Context:    "./app",
				Dockerfile: "Dockerfile.prod",
			},
			expectError: false,
		},
		{
			name:        "missing context",
			build:       Build{},
			expectError: true,
			errorMsg:    "build context is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.build.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVolume_Validate(t *testing.T) {
	tests := []struct {
		name        string
		volume      Volume
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid volume",
			volume: Volume{
				Name: "data-vol",
			},
			expectError: false,
		},
		{
			name: "valid with driver",
			volume: Volume{
				Name:   "my-volume",
				Driver: "local",
			},
			expectError: false,
		},
		{
			name:        "missing name",
			volume:      Volume{},
			expectError: true,
			errorMsg:    "volume name is required",
		},
		{
			name: "invalid name - starts with dash",
			volume: Volume{
				Name: "-invalid",
			},
			expectError: true,
			errorMsg:    "invalid volume name",
		},
		{
			name: "invalid name - special characters",
			volume: Volume{
				Name: "vol@name",
			},
			expectError: true,
			errorMsg:    "invalid volume name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.volume.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNetwork_Validate(t *testing.T) {
	tests := []struct {
		name        string
		network     Network
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid network",
			network: Network{
				Name: "my-network",
			},
			expectError: false,
		},
		{
			name: "valid with driver",
			network: Network{
				Name:   "bridge-net",
				Driver: "bridge",
			},
			expectError: false,
		},
		{
			name:        "missing name",
			network:     Network{},
			expectError: true,
			errorMsg:    "network name is required",
		},
		{
			name: "invalid name - starts with dash",
			network: Network{
				Name: "-invalid",
			},
			expectError: true,
			errorMsg:    "invalid network name",
		},
		{
			name: "invalid name - special characters",
			network: Network{
				Name: "net@work",
			},
			expectError: true,
			errorMsg:    "invalid network name",
		},
		{
			name: "valid with IPAM",
			network: Network{
				Name: "custom-net",
				IPAM: &IPAM{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.network.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIPAM_Validate(t *testing.T) {
	ipam := &IPAM{}
	err := ipam.Validate()
	assert.NoError(t, err)
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already valid",
			input:    "my-service",
			expected: "my-service",
		},
		{
			name:     "with spaces",
			input:    "my service",
			expected: "my-service",
		},
		{
			name:     "with special characters",
			input:    "my@service#test",
			expected: "my-service-test",
		},
		{
			name:     "starts with invalid char",
			input:    "@service",
			expected: "service",
		},
		{
			name:     "ends with invalid char",
			input:    "service@",
			expected: "service",
		},
		{
			name:     "multiple consecutive invalid chars",
			input:    "my@@service",
			expected: "my-service",
		},
		{
			name:     "dots and underscores preserved",
			input:    "my_service.v1",
			expected: "my_service.v1",
		},
		{
			name:     "collapse multiple hyphens",
			input:    "my---service",
			expected: "my-service",
		},
		{
			name:     "mixed invalid chars",
			input:    "!!!my service@@@test###",
			expected: "my-service-test",
		},
		{
			name:     "unicode characters",
			input:    "service™®",
			expected: "service",
		},
		{
			name:     "leading and trailing dashes removed",
			input:    "---service---",
			expected: "service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceNameRegex(t *testing.T) {
	validNames := []string{
		"service",
		"my-service",
		"service_name",
		"service.v1",
		"s1",
		"service123",
		"a",
		"1service",
	}

	for _, name := range validNames {
		t.Run("valid: "+name, func(t *testing.T) {
			assert.True(t, serviceNameRegex.MatchString(name), "expected %q to be valid", name)
		})
	}

	invalidNames := []string{
		"",
		"-service",
		"_service",
		".service",
		"my service",
		"service@name",
		"service#test",
		"service!",
	}

	for _, name := range invalidNames {
		t.Run("invalid: "+name, func(t *testing.T) {
			assert.False(t, serviceNameRegex.MatchString(name), "expected %q to be invalid", name)
		})
	}
}

func TestSpec_Validate_MultipleErrors(t *testing.T) {
	spec := Spec{
		Name: "-invalid-name",
		Container: Container{
			RestartPolicy: "invalid-policy",
		},
		DependsOn: []string{"-invalid-name"},
		Volumes: []Volume{
			{Name: ""},
		},
		Networks: []Network{
			{Name: ""},
		},
	}

	err := spec.Validate()
	require.Error(t, err)

	errStr := err.Error()
	assert.Contains(t, errStr, "invalid service name")
	assert.Contains(t, errStr, "image is required")
	assert.Contains(t, errStr, "service cannot depend on itself")
	assert.Contains(t, errStr, "volume name is required")
	assert.Contains(t, errStr, "network name is required")

	// Should contain multiple errors joined by semicolons
	assert.True(t, strings.Contains(errStr, ";"), "expected multiple errors joined by semicolons")
}
