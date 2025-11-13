package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

// ---------------------------
// ExtractExternalDependencies tests
// ---------------------------

func TestConverter_ExtractExternalDependencies_NoExtension(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.NoError(t, err)
	assert.Nil(t, deps)
}

func TestConverter_ExtractExternalDependencies_SingleRequiredDep(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": []interface{}{
				map[string]interface{}{
					"project": "infrastructure",
					"service": "proxy",
				},
			},
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.NoError(t, err)
	require.Len(t, deps, 1)

	want := []service.ExternalDependency{
		{
			Project:  "infrastructure",
			Service:  "proxy",
			Optional: false,
		},
	}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("external dependencies mismatch (-want +got):\n%s", diff)
	}
}

func TestConverter_ExtractExternalDependencies_OptionalDep(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": []interface{}{
				map[string]interface{}{
					"project":  "monitoring",
					"service":  "prometheus",
					"optional": true,
				},
			},
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.NoError(t, err)
	require.Len(t, deps, 1)

	want := []service.ExternalDependency{
		{
			Project:  "monitoring",
			Service:  "prometheus",
			Optional: true,
		},
	}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("external dependencies mismatch (-want +got):\n%s", diff)
	}
}

func TestConverter_ExtractExternalDependencies_MultipleDeps(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "backend",
		Image: "myapp:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": []interface{}{
				map[string]interface{}{
					"project": "infrastructure",
					"service": "proxy",
				},
				map[string]interface{}{
					"project":  "monitoring",
					"service":  "prometheus",
					"optional": true,
				},
				map[string]interface{}{
					"project": "data",
					"service": "redis-cache",
				},
			},
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.NoError(t, err)
	require.Len(t, deps, 3)

	want := []service.ExternalDependency{
		{
			Project:  "infrastructure",
			Service:  "proxy",
			Optional: false,
		},
		{
			Project:  "monitoring",
			Service:  "prometheus",
			Optional: true,
		},
		{
			Project:  "data",
			Service:  "redis-cache",
			Optional: false,
		},
	}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("external dependencies mismatch (-want +got):\n%s", diff)
	}
}

// ---------------------------
// Error cases
// ---------------------------

func TestConverter_ExtractExternalDependencies_NotAList(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": "should-be-a-list",
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a list")
	assert.Nil(t, deps)
}

func TestConverter_ExtractExternalDependencies_ItemNotAMap(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": []interface{}{
				"not-a-map",
			},
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a map with 'project' and 'service' keys")
	assert.Nil(t, deps)
}

func TestConverter_ExtractExternalDependencies_MissingProject(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": []interface{}{
				map[string]interface{}{
					"service": "proxy",
				},
			},
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify both 'project' and 'service'")
	assert.Nil(t, deps)
}

func TestConverter_ExtractExternalDependencies_MissingService(t *testing.T) {
	composeService := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-depends-on": []interface{}{
				map[string]interface{}{
					"project": "infrastructure",
				},
			},
		},
	}

	converter := NewConverter("/test")
	deps, err := converter.ExtractExternalDependencies(composeService)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify both 'project' and 'service'")
	assert.Nil(t, deps)
}

func TestConverter_ExtractExternalDependencies_InvalidProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
	}{
		{name: "uppercase", projectName: "Infrastructure"},
		{name: "special chars", projectName: "infra!"},
		{name: "spaces", projectName: "my infra"},
		{name: "starts with dash", projectName: "-infra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composeService := types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				Extensions: map[string]interface{}{
					"x-quad-ops-depends-on": []interface{}{
						map[string]interface{}{
							"project": tt.projectName,
							"service": "proxy",
						},
					},
				},
			}

			converter := NewConverter("/test")
			deps, err := converter.ExtractExternalDependencies(composeService)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid project name")
			assert.Nil(t, deps)
		})
	}
}

func TestConverter_ExtractExternalDependencies_InvalidServiceName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
	}{
		{name: "special chars", serviceName: "proxy!"},
		{name: "spaces", serviceName: "my proxy"},
		{name: "starts with dash", serviceName: "-proxy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composeService := types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				Extensions: map[string]interface{}{
					"x-quad-ops-depends-on": []interface{}{
						map[string]interface{}{
							"project": "infrastructure",
							"service": tt.serviceName,
						},
					},
				},
			}

			converter := NewConverter("/test")
			deps, err := converter.ExtractExternalDependencies(composeService)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid service name")
			assert.Nil(t, deps)
		})
	}
}

// ---------------------------
// Integration with ConvertProject
// ---------------------------

func TestConverter_ConvertProject_WithExternalDependencies(t *testing.T) {
	project := &types.Project{
		Name:       "app",
		WorkingDir: "/test",
		Services: types.Services{
			"backend": {
				Name:  "backend",
				Image: "myapp:latest",
				Extensions: map[string]interface{}{
					"x-quad-ops-depends-on": []interface{}{
						map[string]interface{}{
							"project": "infrastructure",
							"service": "proxy",
						},
						map[string]interface{}{
							"project":  "monitoring",
							"service":  "prometheus",
							"optional": true,
						},
					},
				},
			},
		},
	}

	converter := NewConverter("/test")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "app_backend", spec.Name)
	require.Len(t, spec.ExternalDependencies, 2)

	want := []service.ExternalDependency{
		{
			Project:  "infrastructure",
			Service:  "proxy",
			Optional: false,
		},
		{
			Project:  "monitoring",
			Service:  "prometheus",
			Optional: true,
		},
	}
	if diff := cmp.Diff(want, spec.ExternalDependencies); diff != "" {
		t.Errorf("external dependencies mismatch (-want +got):\n%s", diff)
	}
}

func TestConverter_ConvertProject_WithBothIntraAndExternalDeps(t *testing.T) {
	project := &types.Project{
		Name:       "app",
		WorkingDir: "/test",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				DependsOn: map[string]types.ServiceDependency{
					"api": {},
				},
				Extensions: map[string]interface{}{
					"x-quad-ops-depends-on": []interface{}{
						map[string]interface{}{
							"project": "infrastructure",
							"service": "proxy",
						},
					},
				},
			},
			"api": {
				Name:  "api",
				Image: "myapp:latest",
			},
		},
	}

	converter := NewConverter("/test")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)

	var webSpec *service.Spec
	for i := range specs {
		if specs[i].Name == "app_web" {
			webSpec = &specs[i]
			break
		}
	}
	require.NotNil(t, webSpec)

	// Should have intra-project dependency
	assert.Equal(t, []string{"app_api"}, webSpec.DependsOn)

	// Should have external dependency
	want := []service.ExternalDependency{
		{
			Project:  "infrastructure",
			Service:  "proxy",
			Optional: false,
		},
	}
	if diff := cmp.Diff(want, webSpec.ExternalDependencies); diff != "" {
		t.Errorf("external dependencies mismatch (-want +got):\n%s", diff)
	}
}
