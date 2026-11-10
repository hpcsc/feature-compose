package catalog

import (
	"fmt"
	"maps"
	"slices"
)

type optionValues struct {
	byName   map[string]string
	given    map[string]bool
	answered map[string]bool
	defaults map[string]string
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
		lines = append(lines, fmt.Sprintf("%s%s: %s (%s)", prefix, optionName, o.byName[optionName], o.source(optionName)))
	}
	return lines
}

func (o optionValues) withoutValue() []string {
	var optionNames []string
	for _, optionName := range slices.Sorted(maps.Keys(o.byName)) {
		if o.byName[optionName] == "" && o.defaults[optionName] == "" {
			optionNames = append(optionNames, optionName)
		}
	}
	return optionNames
}

func (o optionValues) source(optionName string) string {
	switch {
	case o.given[optionName]:
		return "command line"
	case o.answered[optionName]:
		return "answered"
	}
	return "default"
}
