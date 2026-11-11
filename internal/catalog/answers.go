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
		if q.Feature == "" {
			a.StarterValues[q.Option] = values[i]
			continue
		}
		if a.FeatureValues[q.Feature] == nil {
			a.FeatureValues[q.Feature] = map[string]string{}
		}
		a.FeatureValues[q.Feature][q.Option] = values[i]
	}
	return a
}
