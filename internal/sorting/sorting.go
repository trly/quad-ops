// Package sorting provides utility functions for operations like sorting and iterating over slices and maps
package sorting

import (
	"sort"
)

// SliceProcessor is a function that processes a string item.
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

// SortStringSlice sorts a string slice in-place for deterministic order.
func SortStringSlice(slice []string) {
	if len(slice) > 0 {
		sort.Strings(slice)
	}
}

// GetSortedMapKeys returns a sorted slice of keys from a map for deterministic iteration.
func GetSortedMapKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}

	// For this function, simple allocation is more efficient since we need to return the slice
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
