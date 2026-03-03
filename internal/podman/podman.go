// Package podman wraps podman CLI operations for testability.
package podman

import (
	"fmt"
	"os"
	"os/exec"
)

// PullImages pulls the given container images using podman.
func PullImages(images []string, verbose bool) error {
	if len(images) == 0 {
		return nil
	}

	total := len(images)
	if verbose {
		fmt.Printf("Pulling %d image(s)...\n", total)
	}

	for i, image := range images {
		if verbose {
			fmt.Printf("  [%d/%d] Pulling %s\n", i+1, total, image)
		}

		cmd := exec.Command("podman", "pull", image) //nolint:gosec // image names from validated compose files
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to pull image %s: %w", image, err)
			}
		} else {
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to pull image %s: %w\n%s", image, err, string(output))
			}
		}
	}

	return nil
}
