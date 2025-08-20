package compose

import (
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/unit"
)

// processNetworks processes all networks from a Docker Compose project.
func (p *Processor) processNetworks(project *types.Project) error {
	for networkName, networkConfig := range project.Networks {
		p.logger.Debug("Processing network", "network", networkName)

		// Skip external networks - they are managed externally and should not be created by quad-ops
		if IsExternal(networkConfig.External) {
			p.logger.Debug("Skipping external network", "network", networkName)
			continue
		}

		// Create prefixed network name using project name for consistency
		prefixedName := Prefix(project.Name, networkName)
		network := unit.NewNetwork(prefixedName)
		network = network.FromComposeNetwork(networkName, networkConfig)

		// Use quad-ops preferred naming (no systemd- prefix)
		network.NetworkName = prefixedName

		// Create the quadlet unit
		quadletUnit := unit.QuadletUnit{
			Name:    prefixedName,
			Type:    "network",
			Network: *network,
		}

		// Process the quadlet unit
		if err := p.processUnit(&quadletUnit); err != nil {
			p.logger.Error("Failed to process network unit", "error", err)
		}
	}
	return nil
}
