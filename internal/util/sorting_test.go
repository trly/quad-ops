package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortAndIterateSlice(t *testing.T) {
	// Test empty slice
	SortAndIterateSlice([]string{}, func(s string) {
		t.Error("Function should not be called for empty slice")
	})

	// Test sorted iteration and non-modification of original
	original := []string{"c", "a", "b"}
	originalCopy := make([]string, len(original))
	copy(originalCopy, original)

	result := []string{}
	SortAndIterateSlice(original, func(s string) {
		result = append(result, s)
	})

	// Check if original slice is unchanged
	assert.Equal(t, originalCopy, original, "Original slice should not be modified")

	// Check if items were processed in sorted order
	assert.Equal(t, []string{"a", "b", "c"}, result, "Items should be processed in sorted order")
}

func TestGetSortedMapKeys(t *testing.T) {
	// Test empty map
	emptyMap := map[string]string{}
	assert.Empty(t, GetSortedMapKeys(emptyMap), "Empty map should return empty slice")

	// Test map with keys
	testMap := map[string]string{
		"c": "value3",
		"a": "value1",
		"b": "value2",
	}

	keys := GetSortedMapKeys(testMap)
	assert.Equal(t, []string{"a", "b", "c"}, keys, "Keys should be returned in sorted order")
}
