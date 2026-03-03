package systemd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
)

func TestWriteUnitsCreatesFiles(t *testing.T) {
	tmpDir := t.TempDir()

	units := []Unit{
		{
			Name: "my-container.container",
			File: testIniFile("Container", map[string]string{"Image": "alpine:latest"}),
		},
		{
			Name: "my-volume.volume",
			File: testIniFile("Volume", map[string]string{"Driver": "local"}),
		},
		{
			Name: "my-network.network",
			File: testIniFile("Network", map[string]string{"Driver": "bridge"}),
		},
	}

	err := WriteUnits(units, tmpDir)
	require.NoError(t, err)

	for _, u := range units {
		path := filepath.Join(tmpDir, u.Name)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	}
}

func TestWriteUnitsCreatesDirectory(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "subdir")

	units := []Unit{
		{
			Name: "test.container",
			File: testIniFile("Container", map[string]string{"Image": "alpine:latest"}),
		},
	}

	err := WriteUnits(units, tmpDir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "test.container"))
	require.NoError(t, err)
}

func TestWriteUnitsEmptySlice(t *testing.T) {
	tmpDir := t.TempDir()

	err := WriteUnits(nil, tmpDir)
	require.NoError(t, err)
}

// testIniFile is a helper to create a test ini.File with a section and keys.
func testIniFile(sectionName string, keys map[string]string) *ini.File {
	file := ini.Empty()
	section, _ := file.NewSection(sectionName)
	for k, v := range keys {
		_, _ = section.NewKey(k, v)
	}
	return file
}
