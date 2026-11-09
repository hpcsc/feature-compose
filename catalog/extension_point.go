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
	declared := slices.Sorted(slices.Values(e.itemFields))
	for i, item := range items {
		fields := slices.Sorted(maps.Keys(item))
		if !slices.Equal(fields, declared) {
			return extensionPoint{}, fmt.Errorf("the item %d that the feature %s adds to the extension point %q has the fields [%s], but the extension point declares the fields [%s]",
				i+1, featureName, e.name, strings.Join(fields, ", "), strings.Join(e.itemFields, ", "))
		}
	}
	e.items = append(slices.Clone(e.items), items...)
	return e, nil
}
