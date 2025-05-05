// Package util provides utility functions for operations like sorting.
package util

import (
	"sort"
)

// SortStringSlice sorts any string slice in-place..
func SortStringSlice(slice []string) {
	if len(slice) > 0 {
		sort.Strings(slice)
	}
}

// SortStringSlices sorts multiple string slices in-place.
func SortStringSlices(slices ...[]string) {
	for _, slice := range slices {
		SortStringSlice(slice)
	}
}

// SortStringMapKeys returns a sorted slice of keys from a string map.
func SortStringMapKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
