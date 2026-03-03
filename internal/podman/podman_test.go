package podman

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullImagesEmptySlice(t *testing.T) {
	result, err := PullImages(nil, nil, false)
	assert.NoError(t, err)
	assert.Empty(t, result.UpdatedDigests)
}

func TestPullImagesEmptyList(t *testing.T) {
	result, err := PullImages([]string{}, nil, false)
	assert.NoError(t, err)
	assert.Empty(t, result.UpdatedDigests)
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
