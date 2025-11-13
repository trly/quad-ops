package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/service"
)

func TestQuadletWriter_BasicFormat(t *testing.T) {
	w := NewQuadletWriter()
	w.Set("Unit", "Description", "Test container")
	w.Set("Container", "Image", "nginx:latest")
	w.Set("Install", "WantedBy", "default.target")

	expected := `[Unit]
Description=Test container

[Container]
Image=nginx:latest

[Install]
WantedBy=default.target
`
	assert.Equal(t, expected, w.String())
}

func TestQuadletWriter_MultipleValues(t *testing.T) {
	w := NewQuadletWriter()
	w.Append("Container", "Environment", "FOO=bar", "BAZ=qux")

	result := w.String()
	assert.Contains(t, result, "Environment=FOO=bar")
	assert.Contains(t, result, "Environment=BAZ=qux")
}

func TestQuadletWriter_SortedValues(t *testing.T) {
	w := NewQuadletWriter()
	// Add in reverse alphabetical order
	w.AppendSorted("Unit", "After", "z-service.service", "a-service.service", "m-service.service")

	result := w.String()

	// Find positions to verify sorted order
	posA := indexOf(result, "After=a-service.service")
	posM := indexOf(result, "After=m-service.service")
	posZ := indexOf(result, "After=z-service.service")

	assert.Less(t, posA, posM, "a should come before m")
	assert.Less(t, posM, posZ, "m should come before z")
}

func TestQuadletWriter_Map(t *testing.T) {
	w := NewQuadletWriter()
	m := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
		"AAA": "zzz",
	}

	w.AppendMap("Container", "Environment", m, func(k, v string) string {
		return k + "=" + v
	})

	result := w.String()

	// Verify sorted order (AAA, BAZ, FOO)
	posAAA := indexOf(result, "Environment=AAA=zzz")
	posBAZ := indexOf(result, "Environment=BAZ=qux")
	posFOO := indexOf(result, "Environment=FOO=bar")

	assert.Less(t, posAAA, posBAZ)
	assert.Less(t, posBAZ, posFOO)
}

func TestQuadletWriter_BooleanYesNo(t *testing.T) {
	w := NewQuadletWriter()
	w.SetBool("Network", "IPv6", true)
	w.SetBool("Network", "Internal", true)

	result := w.String()
	assert.Contains(t, result, "IPv6=yes")
	assert.Contains(t, result, "Internal=yes")
	assert.NotContains(t, result, "true")
	assert.NotContains(t, result, "false")
}

func TestQuadletWriter_EmptyValues(t *testing.T) {
	w := NewQuadletWriter()
	w.Set("Container", "Image", "nginx:latest")
	w.Set("Container", "EmptyKey", "")   // Should be skipped
	w.Append("Container", "Environment") // Empty slice, should be skipped

	result := w.String()
	assert.Contains(t, result, "Image=nginx:latest")
	assert.NotContains(t, result, "EmptyKey")
}

func TestQuadletWriter_SectionOrdering(t *testing.T) {
	w := NewQuadletWriter()

	// Add sections in specific order
	w.Set("Unit", "Description", "Test")
	w.Set("Container", "Image", "nginx:latest")
	w.Set("Service", "Restart", "always")
	w.Set("Install", "WantedBy", "default.target")

	result := w.String()

	// Verify sections appear in the order added
	posUnit := indexOf(result, "[Unit]")
	posContainer := indexOf(result, "[Container]")
	posService := indexOf(result, "[Service]")
	posInstall := indexOf(result, "[Install]")

	assert.Less(t, posUnit, posContainer)
	assert.Less(t, posContainer, posService)
	assert.Less(t, posService, posInstall)
}

// Helper function to find position of substring in string.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ============================================================================
// Phase 1: Quadlet Automatic Dependencies Tests
// ============================================================================

func TestRenderContainer_NamedVolume_UsesQuadletSyntax(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "myproject-web",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeVolume,
					Source: "myproject_data",
					Target: "/app/data",
				},
			},
		},
		Volumes: []service.Volume{
			{
				Name:     "myproject_data",
				External: false,
			},
		},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "Volume=myproject_data.volume:/app/data",
		"named volume should use Quadlet .volume suffix syntax")
	assert.NotContains(t, result, "After=myproject_data.volume",
		"should NOT have manual After= for volume (Quadlet adds automatically)")
	assert.NotContains(t, result, "Requires=myproject_data.volume",
		"should NOT have manual Requires= for volume (Quadlet adds automatically)")
}

func TestRenderContainer_ServiceNetwork_UsesQuadletSyntax(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "myproject-web",
			Network: service.NetworkMode{
				ServiceNetworks: []string{"myproject-backend"},
			},
		},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "Network=myproject-backend.network",
		"service network should use Quadlet .network suffix syntax")
	assert.NotContains(t, result, "After=myproject-backend.network",
		"should NOT have manual After= for network (Quadlet adds automatically)")
	assert.NotContains(t, result, "Requires=myproject-backend.network",
		"should NOT have manual Requires= for network (Quadlet adds automatically)")
}

func TestRenderContainer_ServiceDependency_KeepsManualDependencies(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "myproject-web",
		},
		DependsOn: []string{"myproject-db"},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "After=myproject-db.service",
		"service-to-service dependencies MUST keep manual After=")
	assert.Contains(t, result, "Requires=myproject-db.service",
		"service-to-service dependencies MUST keep manual Requires=")
}

func TestRenderContainer_BuildDependency_UsesQuadletSyntax(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "localhost/myproject-web:latest",
			ContainerName: "myproject-web",
			Build: &service.Build{
				Context:    "/build/context",
				Dockerfile: "Containerfile",
				Tags:       []string{"localhost/myproject-web:latest"},
			},
		},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "Image=myproject-web.build",
		"build should use Quadlet .build reference syntax")
	assert.NotContains(t, result, "After=myproject-web-build.service",
		"should NOT have manual After= for build (Quadlet adds automatically)")
	assert.NotContains(t, result, "Requires=myproject-web-build.service",
		"should NOT have manual Requires= for build (Quadlet adds automatically)")
}

func TestRenderContainer_MultipleNamedVolumes_AllUseQuadletSyntax(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-app",
		Description: "App service",
		Container: service.Container{
			Image:         "app:latest",
			ContainerName: "myproject-app",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeVolume,
					Source: "myproject_data",
					Target: "/data",
				},
				{
					Type:   service.MountTypeVolume,
					Source: "myproject-config",
					Target: "/config",
				},
			},
		},
		Volumes: []service.Volume{
			{Name: "myproject_data", External: false},
			{Name: "myproject-config", External: false},
		},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "Volume=myproject_data.volume:/data")
	assert.Contains(t, result, "Volume=myproject-config.volume:/config")
	assert.NotContains(t, result, "After=myproject_data.volume")
	assert.NotContains(t, result, "After=myproject-config.volume")
	assert.NotContains(t, result, "Requires=myproject_data.volume")
	assert.NotContains(t, result, "Requires=myproject-config.volume")
}

func TestRenderContainer_BindMount_NotAffectedByChanges(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "myproject-web",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "/host/path",
					Target: "/container/path",
				},
			},
		},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "Volume=/host/path:/container/path",
		"bind mounts should use traditional syntax (not Quadlet volume reference)")
	assert.Contains(t, result, "RequiresMountsFor=/host/path",
		"bind mounts should still require host path to exist")
}

func TestRenderContainer_MixedDependencies_OnlyInfrastructureRemoved(t *testing.T) {
	spec := service.Spec{
		Name:        "myproject-web",
		Description: "Web service",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "myproject-web",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeVolume,
					Source: "myproject_data",
					Target: "/data",
				},
			},
			Network: service.NetworkMode{
				ServiceNetworks: []string{"myproject-backend"},
			},
		},
		DependsOn: []string{"myproject-db", "myproject-cache"},
		Volumes: []service.Volume{
			{Name: "myproject_data", External: false},
		},
	}

	result := renderContainer(spec)

	assert.Contains(t, result, "Volume=myproject_data.volume:/data")
	assert.Contains(t, result, "Network=myproject-backend.network")
	assert.NotContains(t, result, "After=myproject_data.volume")
	assert.NotContains(t, result, "After=myproject-backend.network")

	assert.Contains(t, result, "After=myproject-db.service",
		"service dependencies must be kept")
	assert.Contains(t, result, "After=myproject-cache.service",
		"service dependencies must be kept")
	assert.Contains(t, result, "Requires=myproject-db.service")
	assert.Contains(t, result, "Requires=myproject-cache.service")
}
