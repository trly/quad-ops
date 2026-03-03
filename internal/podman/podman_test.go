package podman

import (
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
