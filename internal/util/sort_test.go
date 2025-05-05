package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortStringSlice(t *testing.T) {
	// Test with non-empty slice
	slice := []string{"c", "a", "b"}
	SortStringSlice(slice)
	assert.Equal(t, []string{"a", "b", "c"}, slice)

	// Test with empty slice
	empty := []string{}
	SortStringSlice(empty)
	assert.Equal(t, []string{}, empty)

	// Test with nil slice
	var nilSlice []string
	SortStringSlice(nilSlice)
	assert.Nil(t, nilSlice)
}

func TestSortStringSlices(t *testing.T) {
	slice1 := []string{"c", "a", "b"}
	slice2 := []string{"z", "x", "y"}
	var nilSlice []string
	empty := []string{}

	SortStringSlices(slice1, slice2, nilSlice, empty)

	assert.Equal(t, []string{"a", "b", "c"}, slice1)
	assert.Equal(t, []string{"x", "y", "z"}, slice2)
	assert.Nil(t, nilSlice)
	assert.Equal(t, []string{}, empty)
}

func TestSortStringMapKeys(t *testing.T) {
	// Test with populated map
	m := map[string]string{
		"c": "value3",
		"a": "value1",
		"b": "value2",
	}
	keys := SortStringMapKeys(m)
	assert.Equal(t, []string{"a", "b", "c"}, keys)

	// Test with empty map
	empty := map[string]string{}
	keys = SortStringMapKeys(empty)
	assert.Nil(t, keys)

	// Test with nil map
	var nilMap map[string]string
	keys = SortStringMapKeys(nilMap)
	assert.Nil(t, keys)
}
