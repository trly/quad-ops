package systemd

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteUnits writes each unit to a separate file in the quadlet directory.
func WriteUnits(units []Unit, quadletDir string) error {
	if err := os.MkdirAll(quadletDir, 0o755); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	for _, unit := range units {
		filename := filepath.Join(quadletDir, unit.Name)

		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create unit file %s: %w", filename, err)
		}

		if err := unit.WriteUnit(f); err != nil {
			_ = f.Close()
			return fmt.Errorf("failed to write unit file %s: %w", filename, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close unit file %s: %w", filename, err)
		}
	}

	return nil
}
