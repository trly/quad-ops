package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuadletWriter_BasicFormat(t *testing.T) {
	w := NewQuadletWriter()
	w.Set("Unit", "Description", "Test container")
	w.Set("Container", "Image", "nginx:latest")
	w.Set("Install", "WantedBy", "default.target")

	expected := `[Unit]
Description=Test container

[Container]
Image=nginx:latest

[Install]
WantedBy=default.target
`
	assert.Equal(t, expected, w.String())
}

func TestQuadletWriter_MultipleValues(t *testing.T) {
	w := NewQuadletWriter()
	w.Append("Container", "Environment", "FOO=bar", "BAZ=qux")

	result := w.String()
	assert.Contains(t, result, "Environment=FOO=bar")
	assert.Contains(t, result, "Environment=BAZ=qux")
}

func TestQuadletWriter_SortedValues(t *testing.T) {
	w := NewQuadletWriter()
	// Add in reverse alphabetical order
	w.AppendSorted("Unit", "After", "z-service.service", "a-service.service", "m-service.service")

	result := w.String()

	// Find positions to verify sorted order
	posA := indexOf(result, "After=a-service.service")
	posM := indexOf(result, "After=m-service.service")
	posZ := indexOf(result, "After=z-service.service")

	assert.Less(t, posA, posM, "a should come before m")
	assert.Less(t, posM, posZ, "m should come before z")
}

func TestQuadletWriter_Map(t *testing.T) {
	w := NewQuadletWriter()
	m := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
		"AAA": "zzz",
	}

	w.AppendMap("Container", "Environment", m, func(k, v string) string {
		return k + "=" + v
	})

	result := w.String()

	// Verify sorted order (AAA, BAZ, FOO)
	posAAA := indexOf(result, "Environment=AAA=zzz")
	posBAZ := indexOf(result, "Environment=BAZ=qux")
	posFOO := indexOf(result, "Environment=FOO=bar")

	assert.Less(t, posAAA, posBAZ)
	assert.Less(t, posBAZ, posFOO)
}

func TestQuadletWriter_BooleanYesNo(t *testing.T) {
	w := NewQuadletWriter()
	w.SetBool("Network", "IPv6", true)
	w.SetBool("Network", "Internal", true)

	result := w.String()
	assert.Contains(t, result, "IPv6=yes")
	assert.Contains(t, result, "Internal=yes")
	assert.NotContains(t, result, "true")
	assert.NotContains(t, result, "false")
}

func TestQuadletWriter_EmptyValues(t *testing.T) {
	w := NewQuadletWriter()
	w.Set("Container", "Image", "nginx:latest")
	w.Set("Container", "EmptyKey", "")   // Should be skipped
	w.Append("Container", "Environment") // Empty slice, should be skipped

	result := w.String()
	assert.Contains(t, result, "Image=nginx:latest")
	assert.NotContains(t, result, "EmptyKey")
}

func TestQuadletWriter_SectionOrdering(t *testing.T) {
	w := NewQuadletWriter()

	// Add sections in specific order
	w.Set("Unit", "Description", "Test")
	w.Set("Container", "Image", "nginx:latest")
	w.Set("Service", "Restart", "always")
	w.Set("Install", "WantedBy", "default.target")

	result := w.String()

	// Verify sections appear in the order added
	posUnit := indexOf(result, "[Unit]")
	posContainer := indexOf(result, "[Container]")
	posService := indexOf(result, "[Service]")
	posInstall := indexOf(result, "[Install]")

	assert.Less(t, posUnit, posContainer)
	assert.Less(t, posContainer, posService)
	assert.Less(t, posService, posInstall)
}

// Helper function to find position of substring in string.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
