package compose

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/sorting"
)

// LabelConverter converts compose-style labels to unit labels.
func LabelConverter(labels types.Labels) []string {
	if len(labels) > 0 {
		return labels.AsList()
	}
	return nil
}

// OptionsConverter converts driver options to unit options.
func OptionsConverter(opts map[string]string) []string {
	if len(opts) == 0 {
		return nil
	}

	options := make([]string, 0, len(opts))
	for k, v := range opts {
		options = append(options, fmt.Sprintf("%s=%s", k, v))
	}

	// Sort for deterministic order
	sorting.SortStringSlice(options)
	return options
}

// NameResolver resolves resource names from compose configs.
func NameResolver(definedName, keyName string) string {
	if definedName != "" {
		return definedName
	}
	return keyName
}
