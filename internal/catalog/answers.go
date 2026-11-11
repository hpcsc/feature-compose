package catalog

import "maps"

type Answers struct {
	Features      []string
	StarterValues map[string]string
	FeatureValues map[string]map[string]string
}

func (a Answers) WithValue(featureName, optionName, value string) Answers {
	if featureName == "" {
		a.StarterValues = maps.Clone(a.StarterValues)
		if a.StarterValues == nil {
			a.StarterValues = map[string]string{}
		}
		a.StarterValues[optionName] = value
		return a
	}
	featureValues := maps.Clone(a.FeatureValues[featureName])
	if featureValues == nil {
		featureValues = map[string]string{}
	}
	featureValues[optionName] = value
	a.FeatureValues = maps.Clone(a.FeatureValues)
	if a.FeatureValues == nil {
		a.FeatureValues = map[string]map[string]string{}
	}
	a.FeatureValues[featureName] = featureValues
	return a
}
