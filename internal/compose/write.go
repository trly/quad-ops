package compose

import (
	"fmt"

	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/unit"
)

// processUnit handles the processing of a single quadlet unit.
func (p *Processor) processUnit(unitItem *unit.QuadletUnit) error {
	// Track this unit as processed
	unitKey := fmt.Sprintf("%s.%s", unitItem.Name, unitItem.Type)
	p.processedUnits[unitKey] = true

	// Generate unit content
	content := unit.GenerateQuadletUnit(*unitItem)

	// Get unit file path
	unitPath := p.fs.GetUnitFilePath(unitItem.Name, unitItem.Type)

	// Check if unit file content has changed
	hasChanged := p.fs.HasUnitChanged(unitPath, content)

	// Check for potential naming conflicts with existing units
	hasNamingConflict := HasNamingConflict(p.repo, unitItem.Name, unitItem.Type)

	// If forcing update or content has changed or there's a naming conflict, write the file
	if p.force || hasChanged || hasNamingConflict {
		// When verbose, log that a change was detected
		if hasChanged {
			p.logger.Debug("Unit content has changed", "name", unitItem.Name, "type", unitItem.Type)
		} else if hasNamingConflict {
			p.logger.Debug("Unit naming scheme has changed", "name", unitItem.Name, "type", unitItem.Type)
		} else {
			p.logger.Debug("Force updating unit", "name", unitItem.Name, "type", unitItem.Type)
		}

		// Write the file
		if err := p.fs.WriteUnitFile(unitPath, content); err != nil {
			return fmt.Errorf("writing unit file for %s: %w", unitItem.Name, err)
		}

		// Track unit in repository
		if err := p.updateUnitDatabase(unitItem, content); err != nil {
			return fmt.Errorf("tracking unit for %s: %w", unitItem.Name, err)
		}

		// Add to changed units list for restart
		p.changedUnits = append(p.changedUnits, *unitItem)
	} else {
		// Even when the file hasn't changed, we still track the unit
		// to ensure the unit's existence is recorded, but we don't add it to changedUnits
		if err := p.updateUnitDatabase(unitItem, content); err != nil {
			return fmt.Errorf("tracking unit for %s: %w", unitItem.Name, err)
		}
	}

	return nil
}

// updateUnitDatabase updates the unit in the repository.
func (p *Processor) updateUnitDatabase(unitItem *unit.QuadletUnit, content string) error {
	// In the systemd-based approach, we don't need to store unit data
	// The repository handles inferring unit information from filesystem and systemd
	// This function is kept for compatibility but doesn't perform actual database operations

	p.logger.Debug("Tracking unit", "name", unitItem.Name, "type", unitItem.Type)

	// Create a unit record for compatibility (repository will handle it appropriately)
	contentHash := p.fs.GetContentHash(content)
	_, err := p.repo.Create(&repository.Unit{
		Name:     unitItem.Name,
		Type:     unitItem.Type,
		SHA1Hash: []byte(contentHash),
	})
	return err
}
