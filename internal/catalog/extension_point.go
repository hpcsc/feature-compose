package catalog

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type extensionPoint struct {
	name       string
	declarer   string
	itemFields []string
	items      []map[string]string
}

func (e extensionPoint) add(featureName string, items []map[string]string) (extensionPoint, error) {
	for i, item := range items {
		if err := e.checkItem(featureName, i+1, item); err != nil {
			return extensionPoint{}, err
		}
	}
	e.items = append(slices.Clone(e.items), items...)
	return e, nil
}

func (e extensionPoint) checkItem(featureName string, number int, item map[string]string) error {
	fields := slices.Sorted(maps.Keys(item))
	if !slices.Equal(fields, slices.Sorted(slices.Values(e.itemFields))) {
		return fmt.Errorf("the item %d that the feature %s adds to the extension point %q has the fields [%s], but the extension point declares the fields [%s]",
			number, featureName, e.name, strings.Join(fields, ", "), strings.Join(e.itemFields, ", "))
	}
	return nil
}

func (e extensionPoint) values(field string) ([]string, error) {
	if !slices.Contains(e.itemFields, field) {
		return nil, fmt.Errorf("unknown field %q: the extension point %q declares the fields [%s]", field, e.name, strings.Join(e.itemFields, ", "))
	}
	values := make([]string, 0, len(e.items))
	for _, item := range e.items {
		values = append(values, item[field])
	}
	return values, nil
}
