package catalog

import (
	"fmt"
	"maps"
	"slices"
)

type optionValues struct {
	byName map[string]string
	given  map[string]bool
}

func (o optionValues) byReference(prefix string) map[string]string {
	byReference := make(map[string]string, len(o.byName))
	for optionName, value := range o.byName {
		byReference[prefix+optionName] = value
	}
	return byReference
}

func (o optionValues) lines(prefix string) []string {
	lines := make([]string, 0, len(o.byName))
	for _, optionName := range slices.Sorted(maps.Keys(o.byName)) {
		source := "default"
		if o.given[optionName] {
			source = "command line"
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s (%s)", prefix, optionName, o.byName[optionName], source))
	}
	return lines
}
