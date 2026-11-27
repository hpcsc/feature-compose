package catalog

const whenKey = "when"

type item struct {
	fields      map[string]string
	featureName string
	when        expression
}

func newItem(featureName string, declared map[string]string) (item, error) {
	fields := make(map[string]string, len(declared))
	for name, value := range declared {
		if name != whenKey {
			fields[name] = value
		}
	}
	added := item{fields: fields, featureName: featureName}
	if declared[whenKey] == "" {
		return added, nil
	}
	when, err := newExpression(declared[whenKey])
	if err != nil {
		return added, err
	}
	added.when = when
	return added, nil
}

func (i item) owner() string {
	if i.featureName == "" {
		return "the starter"
	}
	return "the feature " + i.featureName
}
