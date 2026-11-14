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
	file       string
	itemFields []string
	items      []item
}

func (e extensionPoint) add(items []item) (extensionPoint, error) {
	for i, added := range items {
		if err := e.checkItem(i+1, added); err != nil {
			return extensionPoint{}, err
		}
	}
	e.items = append(slices.Clone(e.items), items...)
	return e, nil
}

func (e extensionPoint) hasItemFields(fields []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(fields)), slices.Sorted(slices.Values(e.itemFields)))
}

func (e extensionPoint) checkItem(number int, added item) error {
	fields := slices.Sorted(maps.Keys(added.fields))
	if !e.hasItemFields(fields) {
		return fmt.Errorf("the item %d that the feature %s adds to the extension point %q has the fields [%s], but the extension point declares the fields [%s]",
			number, added.featureName, e.name, strings.Join(fields, ", "), strings.Join(e.itemFields, ", "))
	}
	return nil
}

func (e extensionPoint) whenFieldMistakes() []string {
	if !slices.Contains(e.itemFields, whenKey) {
		return nil
	}
	return []string{fmt.Sprintf("the extension point %q declares the item field when, which an item of add_to uses for its own when", e.name)}
}

func (e extensionPoint) fileMistakes(templates []template) []string {
	if e.file == "" {
		return []string{fmt.Sprintf("the extension point %q names no file", e.name)}
	}
	var mistakes []string
	found := false
	for _, t := range templates {
		if t.pathWithoutTmpl() != e.file {
			continue
		}
		found = true
		parsed, err := t.calls()
		if err != nil || slices.Contains(parsed.argumentsOf("items"), e.name) {
			continue
		}
		mistakes = append(mistakes, fmt.Sprintf("%s does not call items %q", t.descriptionInOwner(), e.name))
	}
	if !found {
		return []string{fmt.Sprintf("the file %q of the extension point %q does not exist", e.file, e.name)}
	}
	return mistakes
}

func (e extensionPoint) written(readableOf func(featureName string) map[string]string) ([]map[string]string, error) {
	written := make([]map[string]string, 0, len(e.items))
	for _, added := range e.items {
		ok, err := added.when.trueFor(readableOf(added.featureName))
		if err != nil {
			return nil, fmt.Errorf("the when %q of an item that the feature %s adds to the extension point %q: %w", added.when.text, added.featureName, e.name, err)
		}
		if ok {
			written = append(written, added.fields)
		}
	}
	return written, nil
}

func (e extensionPoint) values(field string, readableOf func(featureName string) map[string]string) ([]string, error) {
	if !slices.Contains(e.itemFields, field) {
		return nil, fmt.Errorf("unknown field %q: the extension point %q declares the fields [%s]", field, e.name, strings.Join(e.itemFields, ", "))
	}
	written, err := e.written(readableOf)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(written))
	for _, fields := range written {
		values = append(values, fields[field])
	}
	return values, nil
}
