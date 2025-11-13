package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestSystemdUnitNameForService(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		svc      string
		expected string
	}{
		{
			name:     "basic service name",
			project:  "infra",
			svc:      "db",
			expected: "infra_db.service",
		},
		{
			name:     "service with hyphen",
			project:  "my-app",
			svc:      "api-server",
			expected: "my-app_api-server.service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := systemdUnitNameForService(tt.project, tt.svc)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLaunchdLabelForService(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		svc      string
		expected string
	}{
		{
			name:     "basic service name",
			project:  "infra",
			svc:      "db",
			expected: "com.quad-ops.infra.db",
		},
		{
			name:     "service with hyphen",
			project:  "my-app",
			svc:      "api-server",
			expected: "com.quad-ops.my-app.api-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := launchdLabelForService(tt.project, tt.svc)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestValidateExternalDependencies_AllExist(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "db", Optional: false},
			},
		},
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, name string) (bool, error) {
			assert.Equal(t, "infra_db.service", name)
			return true, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.NoError(t, err)

	// Verify ExistsInRuntime flag was set
	assert.True(t, specs[0].ExternalDependencies[0].ExistsInRuntime)
}

func TestValidateExternalDependencies_RequiredMissing(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "api", Optional: false},
			},
		},
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required external services not found")
	assert.Contains(t, err.Error(), "infra_api")
	assert.Contains(t, err.Error(), "ensure dependency projects are deployed first")

	// Verify ExistsInRuntime flag was set to false
	assert.False(t, specs[0].ExternalDependencies[0].ExistsInRuntime)
}

func TestValidateExternalDependencies_OptionalMissing(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "monitoring", Service: "prometheus", Optional: true},
			},
		},
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.NoError(t, err)

	// Verify ExistsInRuntime flag was set to false
	assert.False(t, specs[0].ExternalDependencies[0].ExistsInRuntime)
}

func TestValidateExternalDependencies_BatchAware(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "infra_db",
		},
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "db", Optional: false},
			},
		},
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, _ string) (bool, error) {
			t.Fatalf("ExistsFunc should not be called for batch-satisfied dependency")
			return false, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.NoError(t, err)

	// Verify ExistsInRuntime flag was set to true (satisfied by batch)
	assert.True(t, specs[1].ExternalDependencies[0].ExistsInRuntime)
}

func TestValidateExternalDependencies_PlatformAwareSystemd(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "proxy", Optional: false},
			},
		},
	}

	var checkedName string
	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, name string) (bool, error) {
			checkedName = name
			return true, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.NoError(t, err)

	// Verify systemd-specific unit name was used
	assert.Equal(t, "infra_proxy.service", checkedName)
}

func TestValidateExternalDependencies_PlatformAwareLaunchd(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "proxy", Optional: false},
			},
		},
	}

	var checkedName string
	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, name string) (bool, error) {
			checkedName = name
			return true, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "launchd")
	require.NoError(t, err)

	// Verify launchd-specific label was used
	assert.Equal(t, "com.quad-ops.infra.proxy", checkedName)
}

func TestValidateExternalDependencies_MultipleServices(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "db", Optional: false},
				{Project: "infra", Service: "cache", Optional: false},
			},
		},
		{
			Name: "app-worker",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "queue", Optional: false},
			},
		},
	}

	existingServices := map[string]bool{
		"infra-db.service":    true,
		"infra-cache.service": true,
		"infra-queue.service": false, // Missing
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, name string) (bool, error) {
			return existingServices[name], nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required external services not found")
	assert.Contains(t, err.Error(), "infra_queue")
}

func TestValidateExternalResources_AllExist(t *testing.T) {
	ctx := context.Background()

	specs := []service.Spec{
		{
			Name: "app-web",
			Networks: []service.Network{
				{Name: "shared-net", External: true},
			},
			Volumes: []service.Volume{
				{Name: "shared-data", External: true},
			},
		},
	}

	mockRunner := &MockCommandRunner{
		CombinedOutputFunc: func(_ context.Context, name string, args ...string) ([]byte, error) {
			// Expect podman network inspect shared-net
			// Expect podman volume inspect shared-data
			if name == "podman" && len(args) >= 2 {
				if args[0] == "network" && args[1] == "inspect" {
					assert.Equal(t, "shared-net", args[2])
					return nil, nil
				}
				if args[0] == "volume" && args[1] == "inspect" {
					assert.Equal(t, "shared-data", args[2])
					return nil, nil
				}
			}
			t.Fatalf("unexpected command: %s %v", name, args)
			return nil, nil
		},
	}

	err := validateExternalResources(ctx, specs, mockRunner)
	require.NoError(t, err)
}

func TestValidateExternalResources_NetworkMissing(t *testing.T) {
	ctx := context.Background()

	specs := []service.Spec{
		{
			Name: "app-web",
			Networks: []service.Network{
				{Name: "shared-net", External: true},
			},
		},
	}

	mockRunner := &MockCommandRunner{
		CombinedOutputFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, assert.AnError // Network doesn't exist
		},
	}

	err := validateExternalResources(ctx, specs, mockRunner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "external networks not found")
	assert.Contains(t, err.Error(), "shared-net")
	assert.Contains(t, err.Error(), "podman network create")
}

func TestValidateExternalResources_VolumeMissing(t *testing.T) {
	ctx := context.Background()

	specs := []service.Spec{
		{
			Name: "app-web",
			Volumes: []service.Volume{
				{Name: "shared-data", External: true},
			},
		},
	}

	mockRunner := &MockCommandRunner{
		CombinedOutputFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, assert.AnError // Volume doesn't exist
		},
	}

	err := validateExternalResources(ctx, specs, mockRunner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "external volumes not found")
	assert.Contains(t, err.Error(), "shared-data")
	assert.Contains(t, err.Error(), "podman volume create")
}

func TestValidateExternalResources_IgnoresNonExternal(t *testing.T) {
	ctx := context.Background()

	specs := []service.Spec{
		{
			Name: "app-web",
			Networks: []service.Network{
				{Name: "internal-net", External: false},
			},
			Volumes: []service.Volume{
				{Name: "local-data", External: false},
			},
		},
	}

	mockRunner := &MockCommandRunner{
		CombinedOutputFunc: func(_ context.Context, name string, args ...string) ([]byte, error) {
			t.Fatalf("should not check non-external resources: %s %v", name, args)
			return nil, nil
		},
	}

	err := validateExternalResources(ctx, specs, mockRunner)
	require.NoError(t, err)
}

func TestValidateExternalResources_MultipleSpecs(t *testing.T) {
	ctx := context.Background()

	specs := []service.Spec{
		{
			Name: "app-web",
			Networks: []service.Network{
				{Name: "shared-net", External: true},
			},
		},
		{
			Name: "app-worker",
			Networks: []service.Network{
				{Name: "shared-net", External: true}, // Same network (should check once)
			},
			Volumes: []service.Volume{
				{Name: "shared-data", External: true},
			},
		},
	}

	checkedResources := make(map[string]int)
	mockRunner := &MockCommandRunner{
		CombinedOutputFunc: func(_ context.Context, name string, args ...string) ([]byte, error) {
			if name == "podman" && len(args) >= 3 {
				resourceType := args[0]
				resourceName := args[2]
				key := resourceType + ":" + resourceName
				checkedResources[key]++
			}
			return nil, nil
		},
	}

	err := validateExternalResources(ctx, specs, mockRunner)
	require.NoError(t, err)

	// Verify each unique resource was checked exactly once
	assert.Equal(t, 1, checkedResources["network:shared-net"])
	assert.Equal(t, 1, checkedResources["volume:shared-data"])
}

func TestValidateExternalDependencies_UnsupportedPlatform(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "db", Optional: false},
			},
		},
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "windows")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform: windows")
}

func TestValidateExternalDependencies_MixedRequiredOptional(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "infra", Service: "db", Optional: false},
				{Project: "monitoring", Service: "prometheus", Optional: true},
				{Project: "infra", Service: "cache", Optional: false},
			},
		},
	}

	existingServices := map[string]bool{
		"infra_db.service":              true,  // Exists
		"monitoring_prometheus.service": false, // Missing but optional
		"infra_cache.service":           false, // Missing and required
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, name string) (bool, error) {
			return existingServices[name], nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "infra_cache")
	assert.NotContains(t, err.Error(), "monitoring-prometheus")

	// Verify flags were set correctly
	deps := specs[0].ExternalDependencies
	assert.True(t, deps[0].ExistsInRuntime)  // db exists
	assert.False(t, deps[1].ExistsInRuntime) // prometheus missing
	assert.False(t, deps[2].ExistsInRuntime) // cache missing
}

func TestValidateExternalDependencies_SortedErrorOutput(t *testing.T) {
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	specs := []service.Spec{
		{
			Name: "app-web",
			ExternalDependencies: []service.ExternalDependency{
				{Project: "zulu", Service: "zebra", Optional: false},
				{Project: "alpha", Service: "apple", Optional: false},
				{Project: "mike", Service: "mango", Optional: false},
			},
		},
	}

	mockLifecycle := &MockLifecycle{
		ExistsFunc: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	err := validateExternalDependencies(ctx, specs, mockLifecycle, logger, "systemd")
	require.Error(t, err)

	// Error should list missing services
	errMsg := err.Error()
	assert.Contains(t, errMsg, "alpha_apple")
	assert.Contains(t, errMsg, "mike_mango")
	assert.Contains(t, errMsg, "zulu_zebra")
}
