// Package podman wraps podman CLI operations for testability.
package podman

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// remoteDigest returns the digest of an image in its remote registry
// using a lightweight HEAD request (no layer data is downloaded).
func remoteDigest(ctx context.Context, image string) (string, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image reference %s: %w", image, err)
	}

	desc, err := remote.Head(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote digest for %s: %w", image, err)
	}

	return desc.Digest.String(), nil
}

// PullResult reports which images were pulled and their new digests.
type PullResult struct {
	// UpdatedDigests maps image references to their new remote digests
	// after a successful pull.
	UpdatedDigests map[string]string
}

// PullImages pulls the given container images using podman, skipping
// images whose stored digest already matches the remote registry.
// The knownDigests map provides previously-stored remote digests keyed
// by image reference.
func PullImages(images []string, knownDigests map[string]string, verbose bool) (*PullResult, error) {
	result := &PullResult{UpdatedDigests: make(map[string]string)}

	if len(images) == 0 {
		return result, nil
	}

	ctx := context.Background()
	total := len(images)

	if verbose {
		fmt.Printf("Checking %d image(s) for updates...\n", total)
	}

	var pulled int
	for i, image := range images {
		if verbose {
			fmt.Printf("  [%d/%d] Checking %s\n", i+1, total, image)
		}

		remoteDig, err := remoteDigest(ctx, image)
		if err != nil {
			if verbose {
				fmt.Printf("    Could not check remote digest, pulling to be safe: %v\n", err)
			}
		} else if knownDigests[image] == remoteDig {
			if verbose {
				fmt.Printf("    Up to date (%s)\n", remoteDig)
			}
			continue
		} else if verbose {
			local := knownDigests[image]
			if local == "" {
				fmt.Printf("    No stored digest, pull required\n")
			} else {
				fmt.Printf("    Digest changed (local=%s, remote=%s)\n", local, remoteDig)
			}
		}

		if verbose {
			fmt.Printf("  [%d/%d] Pulling %s\n", i+1, total, image)
		}

		cmd := exec.Command("podman", "pull", image) //nolint:gosec // image names from validated compose files
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return result, fmt.Errorf("failed to pull image %s: %w", image, err)
			}
		} else {
			output, err := cmd.CombinedOutput()
			if err != nil {
				return result, fmt.Errorf("failed to pull image %s: %w\n%s", image, err, string(output))
			}
		}
		pulled++

		// Record the digest after a successful pull. If we couldn't
		// fetch the remote digest earlier, try again now.
		if remoteDig == "" {
			remoteDig, err = remoteDigest(ctx, image)
			if err != nil {
				if verbose {
					fmt.Printf("    WARNING: could not determine digest after pull: %v\n", err)
				}
				continue
			}
		}
		result.UpdatedDigests[image] = remoteDig
	}

	if verbose {
		fmt.Printf("Pulled %d of %d image(s) (%d already up to date)\n", pulled, total, total-pulled)
	}

	return result, nil
}
