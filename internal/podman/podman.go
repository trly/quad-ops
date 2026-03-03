// Package podman wraps podman CLI operations for testability.
package podman

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// localDigest returns the digest of a locally-available podman image,
// or an empty string when the image does not exist locally.
func localDigest(ctx context.Context, image string) string {
	cmd := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.Digest}}", image) //nolint:gosec // image names from validated compose files
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

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

// needsPull reports whether the image must be pulled. It compares the
// local digest (from podman inspect) against the remote digest (from a
// registry HEAD request). An image needs pulling when it doesn't exist
// locally or when the digests differ.
func needsPull(ctx context.Context, image string, verbose bool) bool {
	local := localDigest(ctx, image)
	if local == "" {
		if verbose {
			fmt.Printf("    Image not found locally, pull required\n")
		}
		return true
	}

	remoteDig, err := remoteDigest(ctx, image)
	if err != nil {
		if verbose {
			fmt.Printf("    Could not check remote digest, pulling to be safe: %v\n", err)
		}
		return true
	}

	if local != remoteDig {
		if verbose {
			fmt.Printf("    Digest changed (local=%s, remote=%s)\n", local, remoteDig)
		}
		return true
	}

	if verbose {
		fmt.Printf("    Up to date (%s)\n", local)
	}
	return false
}

// PullImages pulls the given container images using podman, skipping
// images whose local digest already matches the remote registry.
func PullImages(images []string, verbose bool) error {
	if len(images) == 0 {
		return nil
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

		if !needsPull(ctx, image, verbose) {
			continue
		}

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
		pulled++
	}

	if verbose {
		fmt.Printf("Pulled %d of %d image(s) (%d already up to date)\n", pulled, total, total-pulled)
	}

	return nil
}
