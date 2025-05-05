package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/util"
)

func TestLabelConverter(t *testing.T) {
	tests := []struct {
		name     string
		labels   types.Labels
		expected []string
	}{
		{
			name:     "Empty labels",
			labels:   types.Labels{},
			expected: nil,
		},
		{
			name: "Multiple labels",
			labels: types.Labels{
				"com.example.label1": "value1",
				"com.example.label2": "value2",
			},
			expected: []string{
				"com.example.label1=value1",
				"com.example.label2=value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LabelConverter(tt.labels)

			// Sort both slices for comparison since map iteration order is non-deterministic
			util.SortStringSlice(result)
			util.SortStringSlice(tt.expected)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOptionsConverter(t *testing.T) {
	tests := []struct {
		name     string
		opts     map[string]string
		expected []string
	}{
		{
			name:     "Empty options",
			opts:     map[string]string{},
			expected: nil,
		},
		{
			name: "Multiple options",
			opts: map[string]string{
				"opt1": "val1",
				"opt2": "val2",
			},
			expected: []string{
				"opt1=val1",
				"opt2=val2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OptionsConverter(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNameResolver(t *testing.T) {
	tests := []struct {
		name         string
		definedName  string
		keyName      string
		expectedName string
	}{
		{
			name:         "Use defined name",
			definedName:  "explicit-volume-name",
			keyName:      "volume1",
			expectedName: "explicit-volume-name",
		},
		{
			name:         "Fall back to key name",
			definedName:  "",
			keyName:      "volume1",
			expectedName: "volume1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NameResolver(tt.definedName, tt.keyName)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}
