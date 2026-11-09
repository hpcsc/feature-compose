package catalog

import (
	"maps"
	"slices"
	"strings"
)

type Combination struct {
	StarterName   string
	StarterValues map[string]string
	FeatureValues map[string]map[string]string
}

func (c Combination) String() string {
	words := []string{c.StarterName}
	for _, optionName := range slices.Sorted(maps.Keys(c.StarterValues)) {
		words = append(words, "--option", optionName+"="+c.StarterValues[optionName])
	}
	for _, featureName := range slices.Sorted(maps.Keys(c.FeatureValues)) {
		values := c.FeatureValues[featureName]
		pairs := make([]string, 0, len(values))
		for _, optionName := range slices.Sorted(maps.Keys(values)) {
			pairs = append(pairs, optionName+"="+values[optionName])
		}
		flag := featureName
		if len(pairs) > 0 {
			flag += ":" + strings.Join(pairs, ",")
		}
		words = append(words, "--feature", flag)
	}
	return strings.Join(words, " ")
}
