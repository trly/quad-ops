// Package util provides utility functions for operations like sorting and iterating over slices and maps
package util

import (
	"sort"
	"sync"
)

// SliceProcessor is a function that processes a string item.
type SliceProcessor func(string)

// stringSlicePool is a sync.Pool for reusing string slices
var stringSlicePool = sync.Pool{
	New: func() interface{} {
		// Start with a reasonable initial capacity
		return make([]string, 0, 16)
	},
}

// SortAndIterateSlice sorts a slice and applies a function to each item.
// This centralizes the common pattern of creating a copy of a slice,
// sorting it, and then iterating over the sorted values.
// Uses object pooling to reduce allocations for temporary slices.
func SortAndIterateSlice(slice []string, fn SliceProcessor) {
	if len(slice) == 0 {
		return
	}

	// Get a slice from the pool
	sorted := stringSlicePool.Get().([]string)
	defer stringSlicePool.Put(sorted[:0]) // Reset length but keep capacity

	// Ensure capacity and copy elements
	if cap(sorted) < len(slice) {
		sorted = make([]string, len(slice))
	} else {
		sorted = sorted[:len(slice)]
	}
	copy(sorted, slice)
	sort.Strings(sorted)

	// Process each item in sorted order
	for _, item := range sorted {
		fn(item)
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
