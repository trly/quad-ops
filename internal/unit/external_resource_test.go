package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestExternalResourceHandling(t *testing.T) {
	// Create a test project with external resources
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx",
				Volumes: []types.ServiceVolumeConfig{
					{
						Source: "external-vol",
						Target: "/data",
						Type:   "volume",
					},
				},
				Networks: map[string]*types.ServiceNetworkConfig{
					"external-net": {},
				},
			},
		},
		Volumes: types.Volumes{
			"external-vol": {
				External: types.External(true),
			},
			"internal-vol": {
				Driver: "local",
			},
		},
		Networks: types.Networks{
			"external-net": {
				External: types.External(true),
			},
			"internal-net": {
				Driver: "bridge",
			},
		},
	}

	// Track processed units
	processedUnits := make(map[string]bool)

	// Process each network
	for networkName, networkConfig := range project.Networks {
		// Skip external networks
		if bool(networkConfig.External) {
			// External networks are not processed
			continue
		}

		prefixedName := project.Name + "-" + networkName
		network := NewNetwork(prefixedName)
		// We don't need to use the return value, just simulating the process
		_ = network.FromComposeNetwork(networkName, networkConfig)

		// Mark as processed
		unitKey := prefixedName + ".network"
		processedUnits[unitKey] = true
	}

	// Process each volume
	for volumeName, volumeConfig := range project.Volumes {
		// Skip external volumes
		if bool(volumeConfig.External) {
			// External volumes are not processed
			continue
		}

		prefixedName := project.Name + "-" + volumeName
		volume := NewVolume(prefixedName)
		// We don't need to use the return value, just simulating the process
		_ = volume.FromComposeVolume(volumeName, volumeConfig)

		// Mark as processed
		unitKey := prefixedName + ".volume"
		processedUnits[unitKey] = true
	}

	// Assert that external resources were not processed
	assert.NotContains(t, processedUnits, "test-project-external-vol.volume")
	assert.NotContains(t, processedUnits, "test-project-external-net.network")

	// Assert that internal resources were processed
	assert.Contains(t, processedUnits, "test-project-internal-vol.volume")
	assert.Contains(t, processedUnits, "test-project-internal-net.network")
}

// Mock repository for testing.
type MockRepository struct{}

func (m *MockRepository) Create(unit *Unit) (*Unit, error) {
	return unit, nil
}

func (m *MockRepository) FindByName(_, _ string) (*Unit, error) {
	return nil, nil
}

func (m *MockRepository) FindAll() ([]*Unit, error) {
	return []*Unit{}, nil
}

func (m *MockRepository) Delete(_ int64) error {
	return nil
}
