package util

import (
	"sort"
)

// SliceProcessor is a function that processes a string item
type SliceProcessor func(string)

// SortAndIterateSlice sorts a slice and applies a function to each item.
// This centralizes the common pattern of creating a copy of a slice,
// sorting it, and then iterating over the sorted values.
func SortAndIterateSlice(slice []string, fn SliceProcessor) {
	if len(slice) == 0 {
		return
	}
	
	// Create a copy to avoid modifying the original
	sorted := make([]string, len(slice))
	copy(sorted, slice)
	sort.Strings(sorted)
	
	// Process each item in sorted order
	for _, item := range sorted {
		fn(item)
	}
}

// GetSortedMapKeys returns a sorted slice of keys from a map for deterministic iteration
func GetSortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}