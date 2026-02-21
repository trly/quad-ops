package systemd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
)

func TestWriteUnit(t *testing.T) {
	t.Run("writes unit content to writer", func(t *testing.T) {
		file := ini.Empty(ini.LoadOptions{AllowShadows: true})
		sec, err := file.NewSection("Container")
		require.NoError(t, err)
		_, err = sec.NewKey("Image", "nginx:latest")
		require.NoError(t, err)

		unit := &Unit{Name: "test", File: file}

		var buf bytes.Buffer
		err = unit.WriteUnit(&buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "[Container]")
		assert.Contains(t, output, "Image = nginx:latest")
	})

	t.Run("writes multiple sections", func(t *testing.T) {
		file := ini.Empty(ini.LoadOptions{AllowShadows: true})

		container, err := file.NewSection("Container")
		require.NoError(t, err)
		_, err = container.NewKey("Image", "redis:7")
		require.NoError(t, err)

		service, err := file.NewSection("Service")
		require.NoError(t, err)
		_, err = service.NewKey("Restart", "always")
		require.NoError(t, err)

		unit := &Unit{Name: "redis", File: file}

		var buf bytes.Buffer
		err = unit.WriteUnit(&buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "[Container]")
		assert.Contains(t, output, "Image = redis:7")
		assert.Contains(t, output, "[Service]")
		assert.Contains(t, output, "Restart = always")
	})

	t.Run("writes shadow keys as repeated directives", func(t *testing.T) {
		file := ini.Empty(ini.LoadOptions{AllowShadows: true})
		sec, err := file.NewSection("Container")
		require.NoError(t, err)

		k, err := sec.NewKey("Volume", "/data:/data:rw")
		require.NoError(t, err)
		require.NoError(t, k.AddShadow("/host:/container:ro"))

		unit := &Unit{Name: "test", File: file}

		var buf bytes.Buffer
		err = unit.WriteUnit(&buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Volume = /data:/data:rw")
		assert.Contains(t, output, "Volume = /host:/container:ro")
	})
}
