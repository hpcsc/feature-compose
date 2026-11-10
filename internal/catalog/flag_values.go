package catalog

import (
	"maps"
	"slices"
	"strings"
)

type flagValues map[string]string

func (v flagValues) optionWords() []string {
	var words []string
	for _, pair := range v.pairs() {
		words = append(words, "--option", pair)
	}
	return words
}

func (v flagValues) featureWords(featureName string) []string {
	flag := featureName
	if pairs := v.pairs(); len(pairs) > 0 {
		flag += ":" + strings.Join(pairs, ",")
	}
	return []string{"--feature", flag}
}

func (v flagValues) pairs() []string {
	pairs := make([]string, 0, len(v))
	for _, optionName := range slices.Sorted(maps.Keys(v)) {
		pairs = append(pairs, optionName+"="+v[optionName])
	}
	return pairs
}
