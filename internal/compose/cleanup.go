package compose

import (
	"fmt"
	"os"
	"strconv"
)

// cleanupOrphans removes orphaned units that are no longer managed.
func (p *Processor) cleanupOrphans() error {
	existingUnits, err := p.repo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from filesystem: %w", err)
	}

	for _, unit := range existingUnits {
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)

		// Check if unit is orphaned
		isOrphaned := !p.processedUnits[unitKey]

		if isOrphaned {
			p.logger.Info("Cleaning up orphaned unit", "unit", unitKey)

			// First, stop the unit
			if err := p.systemd.StopUnit(unit.Name, unit.Type); err != nil {
				p.logger.Warn("Error stopping unit during cleanup", "unit", unitKey, "error", err)
			} else {
				p.logger.Debug("Successfully stopped unit during cleanup", "unit", unitKey)
			}

			// Remove the unit file
			unitPath := p.fs.GetUnitFilePath(unit.Name, unit.Type)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					p.logger.Error("Failed to remove unit file", "path", unitPath, "error", err)
				}
			} else {
				p.logger.Debug("Removed unit file", "path", unitPath)
			}

			// Remove from repository tracking
			if err := p.repo.Delete(strconv.FormatInt(unit.ID, 10)); err != nil {
				p.logger.Error("Failed to remove unit tracking", "unit", unitKey, "error", err)
				continue
			}

			p.logger.Info("Successfully cleaned up unit", "unit", unitKey)
		}
	}

	// Reload systemd after we've removed units
	if err := p.systemd.ReloadSystemd(); err != nil {
		p.logger.Error("Error reloading systemd after cleanup", "error", err)
	}

	return nil
}
