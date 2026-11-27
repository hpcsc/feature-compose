package catalog

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type extensionPoint struct {
	name          string
	declarer      string
	file          string
	itemFields    []string
	items         []item
	generatedFile generatedFileFields
}

func newExtensionPoint(name, declarer string, declared ExtensionPointDefinition) extensionPoint {
	return extensionPoint{
		name: name, declarer: declarer, file: declared.File, itemFields: declared.Item,
		generatedFile: generatedFileFields{
			query: declared.query(), after: declared.After, before: declared.Before,
			separator: declared.Separator, insert: declared.Insert,
		},
	}
}

func (e extensionPoint) inGeneratedFile() bool {
	return e.generatedFile.isSet()
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
		return fmt.Errorf("the item %d that %s adds to the extension point %q has the fields [%s], but the extension point declares the fields [%s]",
			number, added.owner(), e.name, strings.Join(fields, ", "), strings.Join(e.itemFields, ", "))
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
	var mistakes []string
	for _, mistake := range e.generatedFile.unusedMistakes() {
		mistakes = append(mistakes, fmt.Sprintf("the extension point %q %s", e.name, mistake))
	}
	if e.file == "" {
		return append(mistakes, fmt.Sprintf("the extension point %q names no file", e.name))
	}
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
		return append(mistakes, fmt.Sprintf("the file %q of the extension point %q does not exist", e.file, e.name))
	}
	return mistakes
}

func (e extensionPoint) written(readableOf func(featureName string) map[string]string) ([]map[string]string, error) {
	items, err := e.writtenItems(readableOf)
	if err != nil {
		return nil, err
	}
	written := make([]map[string]string, 0, len(items))
	for _, added := range items {
		written = append(written, added.fields)
	}
	return written, nil
}

func (e extensionPoint) writtenItems(readableOf func(featureName string) map[string]string) ([]item, error) {
	var written []item
	for _, added := range e.items {
		ok, err := added.when.trueFor(readableOf(added.featureName))
		if err != nil {
			return nil, fmt.Errorf("the when %q of an item that %s adds to the extension point %q: %w", added.when.text, added.owner(), e.name, err)
		}
		if ok {
			written = append(written, added)
		}
	}
	return written, nil
}

func (e extensionPoint) generatedFileMistakes(starterFolders folders) []string {
	mistakes := e.generatedFile.mistakes(starterFolders, e.file)
	for i, mistake := range mistakes {
		mistakes[i] = fmt.Sprintf("the extension point %q %s", e.name, mistake)
	}
	return mistakes
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
