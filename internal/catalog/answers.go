package catalog

type Answers struct {
	Features      []string
	StarterValues map[string]string
	FeatureValues map[string]map[string]string
}

func (a Answers) WithValues(questions []Question, values []string) Answers {
	a.StarterValues = map[string]string{}
	a.FeatureValues = map[string]map[string]string{}
	for i, q := range questions {
		if q.feature == "" {
			a.StarterValues[q.Option] = values[i]
			continue
		}
		if a.FeatureValues[q.feature] == nil {
			a.FeatureValues[q.feature] = map[string]string{}
		}
		a.FeatureValues[q.feature][q.Option] = values[i]
	}
	return a
}
