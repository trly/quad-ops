package compose

import (
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/unit"
)

// processVolumes processes all volumes from a Docker Compose project.
func (p *Processor) processVolumes(project *types.Project) error {
	for volumeName, volumeConfig := range project.Volumes {
		p.logger.Debug("Processing volume", "volume", volumeName)

		// Skip external volumes - they are managed externally and should not be created by quad-ops
		if IsExternal(volumeConfig.External) {
			p.logger.Debug("Skipping external volume", "volume", volumeName)
			continue
		}

		// Create prefixed volume name using project name for consistency
		prefixedName := Prefix(project.Name, volumeName)
		volume := unit.NewVolume(prefixedName)
		volume = volume.FromComposeVolume(volumeName, volumeConfig)

		// Use quad-ops preferred naming (no systemd- prefix)
		volume.VolumeName = prefixedName

		// Create the quadlet unit
		quadletUnit := unit.QuadletUnit{
			Name:   prefixedName,
			Type:   "volume",
			Volume: *volume,
		}

		// Process the quadlet unit
		if err := p.processUnit(&quadletUnit); err != nil {
			p.logger.Error("Failed to process volume unit", "error", err)
		}
	}
	return nil
}
