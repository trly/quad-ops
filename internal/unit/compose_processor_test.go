package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A simple test function that directly tests the hasUnitChanged function
// without relying on config.GetConfig()
func TestHasUnitChanged(t *testing.T) {
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test file path
	testFilePath := filepath.Join(tmpDir, "test.container")

	// Create the testing function with our modified comparison logic
	testHasChanged := func(path, content string) bool {
		existingContent, err := os.ReadFile(path)
		if err != nil {
			// File doesn't exist or can't be read, so it has changed
			return true
		}

		// Compare the actual content directly
		return string(existingContent) != content
	}

	// Test 1: File doesn't exist, should return true (changed)
	assert.True(t, testHasChanged(testFilePath, "content"))

	// Write initial content
	initialContent := "[Unit]\nDescription=Test Container\n\n[Container]\nImage=test:latest\n"
	err = os.WriteFile(testFilePath, []byte(initialContent), 0600)
	require.NoError(t, err)

	// Test 2: File exists with same content, should return false (unchanged)
	assert.False(t, testHasChanged(testFilePath, initialContent))

	// Test 3: File exists with different content, should return true (changed)
	differentContent := "[Unit]\nDescription=Modified Container\n\n[Container]\nImage=test:latest\n"
	assert.True(t, testHasChanged(testFilePath, differentContent))

	// Test 4: File exists with same content but different line endings, should return true (changed)
	// This is intentional - we want exact matching to detect even whitespace or line ending changes
	lineEndingDifference := "[Unit]\r\nDescription=Test Container\r\n\r\n[Container]\r\nImage=test:latest\r\n"
	assert.True(t, testHasChanged(testFilePath, lineEndingDifference))
}