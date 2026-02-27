package systemd

import (
	"io"
	"maps"
	"slices"

	"gopkg.in/ini.v1"
)

// WriteUnit serializes a unit file to the given writer using go-ini.
func (u *Unit) WriteUnit(w io.Writer) error {
	_, err := u.File.WriteTo(w)
	return err
}

// writeOrderedSection writes sectionMap and shadowMap to an INI section
// with keys in sorted order for deterministic output.
func writeOrderedSection(section *ini.Section, sectionMap map[string]string, shadowMap map[string][]string) {
	for _, key := range slices.Sorted(maps.Keys(sectionMap)) {
		_, _ = section.NewKey(key, sectionMap[key])
	}

	for _, key := range slices.Sorted(maps.Keys(shadowMap)) {
		values := shadowMap[key]
		if len(values) == 0 {
			continue
		}
		k, _ := section.GetKey(key)
		if k == nil {
			k, _ = section.NewKey(key, values[0])
		} else {
			k.SetValue(values[0])
		}
		for _, v := range values[1:] {
			if err := k.AddShadow(v); err != nil {
				continue
			}
		}
	}
}
