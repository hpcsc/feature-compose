package catalog

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type valuesSetByNeeds struct {
	neededFeature string
	byOption      map[string]string
	needOf        map[string][]string
}

func newValuesSetByNeeds(neededFeature string) valuesSetByNeeds {
	return valuesSetByNeeds{neededFeature: neededFeature, byOption: map[string]string{}, needOf: map[string][]string{}}
}

func (v valuesSetByNeeds) add(needing string, values map[string]string) (valuesSetByNeeds, error) {
	optionNames := slices.Sorted(maps.Keys(values))
	for _, optionName := range optionNames {
		if earlier, ok := v.byOption[optionName]; ok && earlier != values[optionName] {
			return valuesSetByNeeds{}, fmt.Errorf("the option %s of the feature %s has two values: %q from the need of %s and %q from the need of %s",
				optionName, v.neededFeature, earlier, strings.Join(v.needOf[optionName], ", "), values[optionName], needing)
		}
	}
	v.byOption, v.needOf = maps.Clone(v.byOption), maps.Clone(v.needOf)
	for _, optionName := range optionNames {
		v.byOption[optionName] = values[optionName]
		v.needOf[optionName] = append(slices.Clone(v.needOf[optionName]), needing)
	}
	return v, nil
}

func (v valuesSetByNeeds) value(optionName string) (string, []string, bool) {
	value, ok := v.byOption[optionName]
	return value, v.needOf[optionName], ok
}

func (v valuesSetByNeeds) checkGiven(optionName, given string) error {
	value, needOf, ok := v.value(optionName)
	if ok && value != given {
		return fmt.Errorf("the option %s of the feature %s has two values: %q from the command line and %q from the need of %s",
			optionName, v.neededFeature, given, value, strings.Join(needOf, ", "))
	}
	return nil
}
