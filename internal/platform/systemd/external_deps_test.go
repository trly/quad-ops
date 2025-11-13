package systemd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

// ---------------------------
// External Dependencies in [Unit] section
// ---------------------------

func TestQuadletRender_ExternalDependencies_Required(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{
				Project:         "infrastructure",
				Service:         "proxy",
				Optional:        false,
				ExistsInRuntime: true, // TODO(quad-ops-dep6): Validation sets this
			},
		},
	}

	result := renderContainer(spec)

	// Required external deps should have both After= and Requires=
	assert.Contains(t, result, "After=infrastructure_proxy.service")
	assert.Contains(t, result, "Requires=infrastructure_proxy.service")
}

func TestQuadletRender_ExternalDependencies_Optional(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{
				Project:         "monitoring",
				Service:         "prometheus",
				Optional:        true,
				ExistsInRuntime: true, // TODO(quad-ops-dep6): Validation sets this
			},
		},
	}

	result := renderContainer(spec)

	// Optional external deps should have After= only (NOT Wants=, NOT Requires=)
	assert.Contains(t, result, "After=monitoring_prometheus.service")
	assert.NotContains(t, result, "Requires=monitoring_prometheus.service",
		"optional deps should not have Requires=")
	assert.NotContains(t, result, "Wants=monitoring_prometheus.service",
		"optional deps should not have Wants= (don't auto-start)")
}

func TestQuadletRender_ExternalDependencies_OptionalMissing(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{
				Project:         "monitoring",
				Service:         "prometheus",
				Optional:        true,
				ExistsInRuntime: false, // Not deployed
			},
		},
	}

	result := renderContainer(spec)

	// Optional missing deps should still have After= (systemd tolerates missing units with After=)
	assert.Contains(t, result, "After=monitoring_prometheus.service")
	assert.NotContains(t, result, "Requires=monitoring_prometheus.service")
}

func TestQuadletRender_ExternalDependencies_Multiple(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{
				Project:         "infrastructure",
				Service:         "proxy",
				Optional:        false,
				ExistsInRuntime: true,
			},
			{
				Project:         "data",
				Service:         "redis",
				Optional:        false,
				ExistsInRuntime: true,
			},
			{
				Project:         "monitoring",
				Service:         "prometheus",
				Optional:        true,
				ExistsInRuntime: true,
			},
		},
	}

	result := renderContainer(spec)

	// Required deps
	assert.Contains(t, result, "After=infrastructure_proxy.service")
	assert.Contains(t, result, "Requires=infrastructure_proxy.service")
	assert.Contains(t, result, "After=data_redis.service")
	assert.Contains(t, result, "Requires=data_redis.service")

	// Optional deps
	assert.Contains(t, result, "After=monitoring_prometheus.service")
	assert.NotContains(t, result, "Requires=monitoring_prometheus.service")
}

func TestQuadletRender_ExternalDependencies_WithIntraProjectDeps(t *testing.T) {
	spec := service.Spec{
		Name:        "app-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "app-web",
		},
		DependsOn: []string{"app-api"},
		ExternalDependencies: []service.ExternalDependency{
			{
				Project:         "infrastructure",
				Service:         "proxy",
				Optional:        false,
				ExistsInRuntime: true,
			},
		},
	}

	result := renderContainer(spec)

	// Intra-project deps
	assert.Contains(t, result, "After=app-api.service")
	assert.Contains(t, result, "Requires=app-api.service")

	// External deps
	assert.Contains(t, result, "After=infrastructure_proxy.service")
	assert.Contains(t, result, "Requires=infrastructure_proxy.service")
}

func TestQuadletRender_ExternalDependencies_Sorted(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{Project: "z-project", Service: "service"},
			{Project: "a-project", Service: "service"},
			{Project: "m-project", Service: "service"},
		},
	}

	result := renderContainer(spec)

	// Find positions of After= directives
	posA := strings.Index(result, "After=a-project_service.service")
	posM := strings.Index(result, "After=m-project_service.service")
	posZ := strings.Index(result, "After=z-project_service.service")

	require.NotEqual(t, -1, posA, "a-project_service not found")
	require.NotEqual(t, -1, posM, "m-project_service not found")
	require.NotEqual(t, -1, posZ, "z-project_service not found")

	// Verify alphabetical ordering
	assert.Less(t, posA, posM, "a-project should come before m-project")
	assert.Less(t, posM, posZ, "m-project should come before z-project")
}

func TestQuadletRender_ExternalDependencies_PreservesUnderscores(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{
				Project:         "my_project",
				Service:         "api_server",
				Optional:        false,
				ExistsInRuntime: true,
			},
			{
				Project:         "data-layer",
				Service:         "postgres_db",
				Optional:        false,
				ExistsInRuntime: true,
			},
		},
	}

	result := renderContainer(spec)

	// Verify underscores are preserved using Prefix logic (project_service)
	assert.Contains(t, result, "After=my_project_api_server.service")
	assert.Contains(t, result, "Requires=my_project_api_server.service")
	assert.Contains(t, result, "After=data-layer_postgres_db.service")
	assert.Contains(t, result, "Requires=data-layer_postgres_db.service")

	// Verify old hyphen format is NOT used
	assert.NotContains(t, result, "my_project-api_server")
	assert.NotContains(t, result, "data-layer-postgres_db")
}
