package catalog

import (
	"context"
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

func (c Catalog) TestCombination(ctx context.Context, combination Combination, name, output string) error {
	starter, err := c.Starter(combination.StarterName)
	if err != nil {
		return err
	}
	selection, err := starter.Select(name, combination.StarterValues, combination.FeatureValues, Answers{})
	if err != nil {
		return err
	}
	if err := selection.Write(output); err != nil {
		return err
	}
	if err := selection.RunAfterRender(ctx, output); err != nil {
		return err
	}
	return selection.RunTests(ctx, output)
}
