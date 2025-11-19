//go:build darwin

package launchd

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

// ---------------------------
// External Dependencies in DependsOn array
// ---------------------------

func TestLaunchdRender_ExternalDependencies_Required(t *testing.T) {
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

	opts := testOptions()
	plist, deps := buildTestPlist(t, spec, opts)

	// Required external deps should be in DependsOn
	expectedLabel := opts.LabelPrefix + ".infrastructure.proxy"
	assert.Contains(t, deps, expectedLabel)
	assert.Len(t, plist.DependsOn, 1)
}

func TestLaunchdRender_ExternalDependencies_Optional(t *testing.T) {
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
				ExistsInRuntime: true, // Deployed
			},
		},
	}

	opts := testOptions()
	plist, deps := buildTestPlist(t, spec, opts)

	// Optional external deps that exist should be in DependsOn
	expectedLabel := opts.LabelPrefix + ".monitoring.prometheus"
	assert.Contains(t, deps, expectedLabel)
	assert.Len(t, plist.DependsOn, 1)
}

func TestLaunchdRender_ExternalDependencies_OptionalMissing(t *testing.T) {
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

	opts := testOptions()
	plist, deps := buildTestPlist(t, spec, opts)

	// Optional missing deps should NOT be in DependsOn
	unexpectedLabel := opts.LabelPrefix + ".monitoring.prometheus"
	assert.NotContains(t, deps, unexpectedLabel)
	assert.Len(t, plist.DependsOn, 0)
}

func TestLaunchdRender_ExternalDependencies_Multiple(t *testing.T) {
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
			{
				Project:         "logging",
				Service:         "loki",
				Optional:        true,
				ExistsInRuntime: false, // Not deployed
			},
		},
	}

	opts := testOptions()
	plist, deps := buildTestPlist(t, spec, opts)

	// Required deps should be present
	assert.Contains(t, deps, opts.LabelPrefix+".infrastructure.proxy")
	assert.Contains(t, deps, opts.LabelPrefix+".data.redis")

	// Optional deployed deps should be present
	assert.Contains(t, deps, opts.LabelPrefix+".monitoring.prometheus")

	// Optional missing deps should NOT be present
	assert.NotContains(t, deps, opts.LabelPrefix+".logging.loki")

	assert.Len(t, plist.DependsOn, 3)
}

func TestLaunchdRender_ExternalDependencies_WithIntraProjectDeps(t *testing.T) {
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

	opts := testOptions()
	plist, deps := buildTestPlist(t, spec, opts)

	// Intra-project deps
	assert.Contains(t, deps, opts.LabelFor("app-api"))

	// External deps
	assert.Contains(t, deps, opts.LabelPrefix+".infrastructure.proxy")

	assert.Len(t, plist.DependsOn, 2)
}

func TestLaunchdRender_ExternalDependencies_Sorted(t *testing.T) {
	spec := service.Spec{
		Name:        "app-backend",
		Description: "Backend service",
		Container: service.Container{
			Image:         "myapp:latest",
			ContainerName: "app-backend",
		},
		ExternalDependencies: []service.ExternalDependency{
			{Project: "z-project", Service: "service", ExistsInRuntime: true},
			{Project: "a-project", Service: "service", ExistsInRuntime: true},
			{Project: "m-project", Service: "service", ExistsInRuntime: true},
		},
	}

	opts := testOptions()
	plist, _ := buildTestPlist(t, spec, opts)

	// DependsOn should be sorted
	expected := []string{
		opts.LabelPrefix + ".a-project.service",
		opts.LabelPrefix + ".m-project.service",
		opts.LabelPrefix + ".z-project.service",
	}

	// Verify sorted
	assert.Equal(t, expected, plist.DependsOn)
}

// ---------------------------
// Test helpers
// ---------------------------

// buildTestPlist builds a plist for testing and returns both the plist and the DependsOn array.
func buildTestPlist(t *testing.T, spec service.Spec, opts Options) (*Plist, []string) {
	t.Helper()

	renderer, err := NewRenderer(opts, testutil.NewTestLogger(t))
	require.NoError(t, err)

	artifacts, err := renderer.renderService(spec)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	// Parse plist content to verify structure
	content := string(artifacts[0].Content)

	// Extract DependsOn array from plist XML
	deps := extractDependsOn(t, content)

	// Also build the plist directly for field-level assertions
	label := opts.LabelFor(spec.Name)

	// Map intra-project dependencies
	var intraDeps []string
	if len(spec.DependsOn) > 0 {
		intraDeps = make([]string, len(spec.DependsOn))
		for i, depName := range spec.DependsOn {
			intraDeps[i] = opts.LabelFor(depName)
		}
	}

	// Map external dependencies (mimicking renderer logic)
	var externalDeps []string
	if len(spec.ExternalDependencies) > 0 {
		for _, dep := range spec.ExternalDependencies {
			// Skip optional deps that don't exist
			if dep.Optional && !dep.ExistsInRuntime {
				continue
			}
			externalLabel := opts.LabelPrefix + "." + dep.Project + "." + dep.Service
			externalDeps = append(externalDeps, externalLabel)
		}
	}

	// Combine and sort
	allDeps := append(intraDeps, externalDeps...)
	sort.Strings(allDeps)

	plist := &Plist{
		Label:     label,
		DependsOn: allDeps,
	}

	return plist, deps
}

// extractDependsOn extracts DependsOn array from plist XML content.
func extractDependsOn(t *testing.T, content string) []string {
	t.Helper()

	var deps []string

	// Find <key>DependsOn</key> section
	if !strings.Contains(content, "<key>DependsOn</key>") {
		return deps
	}

	// Simple XML parsing for test purposes
	lines := strings.Split(content, "\n")
	inDependsOn := false
	inArray := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "<key>DependsOn</key>" {
			inDependsOn = true
			continue
		}

		if inDependsOn && trimmed == "<array>" {
			inArray = true
			continue
		}

		if inArray && trimmed == "</array>" {
			break
		}

		if inArray && strings.HasPrefix(trimmed, "<string>") {
			// Extract value between <string> and </string>
			value := strings.TrimPrefix(trimmed, "<string>")
			value = strings.TrimSuffix(value, "</string>")
			deps = append(deps, value)
		}
	}

	return deps
}
