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
	words := append([]string{c.StarterName}, flagValues(c.StarterValues).optionWords()...)
	for _, featureName := range slices.Sorted(maps.Keys(c.FeatureValues)) {
		words = append(words, flagValues(c.FeatureValues[featureName]).featureWords(featureName)...)
	}
	return strings.Join(words, " ")
}
