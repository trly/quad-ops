package compose

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/service"
)

var update = flag.Bool("update", false, "update golden files")

// TestBundle represents the complete conversion output.
type TestBundle struct {
	Services []service.Spec    `json:"services"`
	Volumes  []service.Volume  `json:"volumes"`
	Networks []service.Network `json:"networks"`
}

func TestSpecConverter_Golden(t *testing.T) {
	goldenDir := filepath.Join("testdata", "golden")

	cases, err := os.ReadDir(goldenDir)
	require.NoError(t, err, "failed to read golden directory")

	for _, entry := range cases {
		if !entry.IsDir() {
			continue
		}

		caseName := entry.Name()
		t.Run(caseName, func(t *testing.T) {
			caseDir := filepath.Join(goldenDir, caseName)
			composePath := filepath.Join(caseDir, "compose.yml")
			goldenPath := filepath.Join(caseDir, "expected.json")

			// Load compose project using existing reader
			logger := log.NewLogger(false)
			project, err := ParseComposeFileWithLogger(composePath, logger)
			require.NoError(t, err, "failed to load compose project")

			// Convert using SpecConverter
			converter := NewSpecConverter(caseDir)
			specs, err := converter.ConvertProject(project)
			require.NoError(t, err, "failed to convert project")

			// Convert volumes and networks
			volumes := converter.convertProjectVolumes(project)
			networks := converter.convertProjectNetworks(project)

			// Validate all outputs
			for _, spec := range specs {
				assert.NoError(t, spec.Validate(), "spec %s failed validation", spec.Name)
			}
			for _, vol := range volumes {
				assert.NoError(t, vol.Validate(), "volume %s failed validation", vol.Name)
			}
			for _, net := range networks {
				assert.NoError(t, net.Validate(), "network %s failed validation", net.Name)
			}

			// Create normalized bundle
			bundle := normalizeBundle(specs, volumes, networks)

			// Marshal to JSON
			actual, err := json.MarshalIndent(bundle, "", "  ")
			require.NoError(t, err, "failed to marshal bundle")

			// Update golden file if flag is set
			if *update {
				err := os.WriteFile(goldenPath, actual, 0600)
				require.NoError(t, err, "failed to write golden file")
				t.Logf("Updated golden file: %s", goldenPath)
				return
			}

			// Compare with golden file
			// #nosec G304 -- goldenPath is constructed from sanitized test case directory names
			expected, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "failed to read golden file (run with -update to create)")

			assert.JSONEq(t, string(expected), string(actual), "bundle does not match golden file")
		})
	}
}

func normalizeBundle(specs []service.Spec, volumes []service.Volume, networks []service.Network) TestBundle {
	// Sort services by name
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Name < specs[j].Name
	})

	// Normalize each spec
	for i := range specs {
		normalizeSpec(&specs[i])
	}

	// Sort volumes by name
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	// Normalize each volume
	for i := range volumes {
		normalizeVolume(&volumes[i])
	}

	// Sort networks by name
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})

	// Normalize each network
	for i := range networks {
		normalizeNetwork(&networks[i])
	}

	return TestBundle{
		Services: specs,
		Volumes:  volumes,
		Networks: networks,
	}
}

func normalizeSpec(spec *service.Spec) {
	// Sort dependencies
	sort.Strings(spec.DependsOn)

	// Sort ports
	sort.Slice(spec.Container.Ports, func(i, j int) bool {
		pi, pj := spec.Container.Ports[i], spec.Container.Ports[j]
		if pi.Host != pj.Host {
			return pi.Host < pj.Host
		}
		if pi.HostPort != pj.HostPort {
			return pi.HostPort < pj.HostPort
		}
		if pi.Container != pj.Container {
			return pi.Container < pj.Container
		}
		return pi.Protocol < pj.Protocol
	})

	// Sort mounts
	sort.Slice(spec.Container.Mounts, func(i, j int) bool {
		mi, mj := spec.Container.Mounts[i], spec.Container.Mounts[j]
		if mi.Type != mj.Type {
			return mi.Type < mj.Type
		}
		if mi.Target != mj.Target {
			return mi.Target < mj.Target
		}
		return mi.Source < mj.Source
	})

	// Sort security fields
	sort.Strings(spec.Container.Security.CapAdd)
	sort.Strings(spec.Container.Security.CapDrop)

	// Sort ulimits
	sort.Slice(spec.Container.Ulimits, func(i, j int) bool {
		return spec.Container.Ulimits[i].Name < spec.Container.Ulimits[j].Name
	})

	// Sort build args and tags if present
	if spec.Container.Build != nil {
		sort.Strings(spec.Container.Build.Tags)
	}

	// Sort tmpfs
	sort.Strings(spec.Container.Tmpfs)

	// Sort volumes attached to this spec
	sort.Slice(spec.Volumes, func(i, j int) bool {
		return spec.Volumes[i].Name < spec.Volumes[j].Name
	})

	// Sort networks attached to this spec
	sort.Slice(spec.Networks, func(i, j int) bool {
		return spec.Networks[i].Name < spec.Networks[j].Name
	})
}

func normalizeVolume(_ *service.Volume) {
	// Volumes are already simple, nothing special to sort
}

func normalizeNetwork(net *service.Network) {
	// Sort IPAM pools by subnet if present
	if net.IPAM != nil && len(net.IPAM.Config) > 0 {
		sort.Slice(net.IPAM.Config, func(i, j int) bool {
			return net.IPAM.Config[i].Subnet < net.IPAM.Config[j].Subnet
		})
	}
}
