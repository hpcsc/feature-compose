package catalog

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type optionValues struct {
	byName   map[string]string
	given    map[string]bool
	needOf   map[string][]string
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
		if o.missing(optionName) {
			optionNames = append(optionNames, optionName)
		}
	}
	return optionNames
}

func (o optionValues) missing(optionName string) bool {
	return o.byName[optionName] == "" && o.defaults[optionName] == ""
}

func (o optionValues) source(optionName string) string {
	needOf := strings.Join(o.needOf[optionName], ", ")
	switch {
	case o.given[optionName] && needOf != "":
		return "command line, need of " + needOf
	case o.given[optionName]:
		return "command line"
	case needOf != "":
		return "need of " + needOf
	case o.answered[optionName]:
		return "answered"
	}
	return "default"
}

func (o optionValues) valuesOfArguments() map[string]string {
	values := map[string]string{}
	for optionName, value := range o.byName {
		if value != o.defaults[optionName] && len(o.needOf[optionName]) == 0 {
			values[optionName] = value
		}
	}
	return values
}
