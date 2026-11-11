package catalog

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
)

type feature struct {
	name            string
	templates       []template
	rules           []delimitersRule
	options         options
	extensionPoints map[string]extensionPoint
	addTo           map[string][]map[string]string
	needs           []string
}

func readFeatureFor(fsys fs.FS, featureName, starterName string) (feature, bool, error) {
	root := path.Join("features", featureName)
	ymlPath := featureYMLPath(featureName)
	parsed, err := readFeatureYML(fsys, featureName)
	if err != nil {
		return feature{}, false, err
	}
	if !slices.Contains(parsed.Targets, starterName) {
		return feature{}, false, nil
	}
	var rules []delimitersRule
	for _, declared := range parsed.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return feature{}, false, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		rules = append(rules, rule)
	}
	var templates []template
	for _, entry := range parsed.Files {
		entryRoot := path.Join(root, entry.Path)
		if !strings.HasPrefix(entryRoot, root+"/") {
			return feature{}, false, fmt.Errorf("read %s: the files entry %q is outside the folder of the feature", ymlPath, entry.Path)
		}
		read, err := readTemplates(fsys, entryRoot, featureName)
		if err != nil {
			return feature{}, false, err
		}
		templates = append(templates, read...)
	}
	declaredOptions := options{declarer: "the feature " + featureName, declared: make(map[string]option, len(parsed.Options))}
	for optionName, declared := range parsed.Options {
		declaredOptions.declared[optionName], err = newOption(optionName, declared.Default, declared.Pattern, declared.Values)
		if err != nil {
			return feature{}, false, fmt.Errorf("read %s: %w", ymlPath, err)
		}
	}
	extensionPoints := make(map[string]extensionPoint, len(parsed.ExtensionPoints))
	for name, declared := range parsed.ExtensionPoints {
		extensionPoints[name] = extensionPoint{name: name, declarer: "the feature " + featureName, itemFields: declared.Item}
	}
	return feature{name: featureName, templates: templates, rules: rules, options: declaredOptions, extensionPoints: extensionPoints, addTo: parsed.AddTo, needs: parsed.needNames}, true, nil
}

func (f feature) placedIn(folders map[string]folder) ([]template, error) {
	placed := make([]template, 0, len(f.templates))
	for _, t := range f.templates {
		inFolder, err := t.place(folders, f.rules)
		if err != nil {
			return nil, err
		}
		placed = append(placed, inFolder)
	}
	return placed, nil
}

func (f feature) addsToOneExtensionPointWith(other feature) bool {
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		if _, ok := other.addTo[name]; ok {
			return true
		}
	}
	return false
}
