package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDev(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"default dev build", "dev", true},
		{"release version", "1.2.3", false},
		{"empty string", "", false},
		{"prefix match is not dev", "develop", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := Version
			t.Cleanup(func() { Version = original })

			Version = tt.version
			assert.Equal(t, tt.want, IsDev())
		})
	}
}

func TestInitPopulatesBuildInfo(t *testing.T) {
	// init() runs before tests; when built with `go test`, ReadBuildInfo
	// succeeds and populates GoVersion from the toolchain.
	assert.NotEmpty(t, GoVersion)
	assert.NotEqual(t, "unknown", GoVersion)
}

func TestUpdateStatusZeroValue(t *testing.T) {
	var status UpdateStatus
	assert.False(t, status.Available)
	assert.Empty(t, status.NewVersion)
	assert.Empty(t, status.AssetURL)
	assert.Empty(t, status.AssetName)
}
