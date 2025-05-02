package unit

import (
	"crypto/sha1"
	"fmt"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicUnitContent(t *testing.T) {
	// Create two identical container configs
	container1 := NewContainer("test-container")
	container2 := NewContainer("test-container")

	// Configure them identically
	service := types.ServiceConfig{
		Image: "docker.io/test:latest",
		Environment: types.MappingWithEquals{
			"VAR1": &[]string{"value1"}[0],
			"VAR2": &[]string{"value2"}[0],
		},
	}

	// Convert to container objects
	container1.FromComposeService(service, "test-project")
	// Do it again for the second container
	container2.FromComposeService(service, "test-project")

	// Generate quadlet units
	unit1 := &QuadletUnit{
		Name:      "test-container",
		Type:      "container",
		Container: *container1,
	}

	unit2 := &QuadletUnit{
		Name:      "test-container",
		Type:      "container",
		Container: *container2,
	}

	// Generate the content - this should be deterministic
	content1 := GenerateQuadletUnit(*unit1, false)
	content2 := GenerateQuadletUnit(*unit2, false)

	// Should produce identical content
	assert.Equal(t, content1, content2, "Unit content should be identical when generated from identical configs")

	// Check if content hashing is deterministic
	hash1 := GetContentHash(content1)
	hash2 := GetContentHash(content2)

	assert.Equal(t, hash1, hash2, "Content hashes should be identical for identical content")
}

func GetContentHash(content string) string {
	// Import the crypto/sha1 package
	hash := sha1.New() //nolint:gosec // Not used for security purposes
	hash.Write([]byte(content))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
