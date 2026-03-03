package podman

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullImagesEmptySlice(t *testing.T) {
	err := PullImages(nil, false)
	assert.NoError(t, err)
}

func TestPullImagesEmptyList(t *testing.T) {
	err := PullImages([]string{}, false)
	assert.NoError(t, err)
}

func TestLocalDigestMissingImage(t *testing.T) {
	digest := localDigest(context.Background(), "nonexistent-image-abc123xyz:latest")
	assert.Empty(t, digest, "non-existent image should return empty digest")
}

func TestRemoteDigestInvalidReference(t *testing.T) {
	_, err := remoteDigest(context.Background(), "://invalid")
	assert.Error(t, err, "invalid image reference should return an error")
}

func TestRemoteDigestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := remoteDigest(ctx, "docker.io/library/alpine:latest")
	assert.Error(t, err, "cancelled context should return an error")
}

func TestNeedsPullMissingLocalImage(t *testing.T) {
	result := needsPull(context.Background(), "nonexistent-image-abc123xyz:latest", false)
	assert.True(t, result, "non-existent local image should need pulling")
}
