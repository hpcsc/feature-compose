package catalog

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

type feature struct {
	name            string
	templates       []template
	rules           []delimitersRule
	options         options
	extensionPoints map[string]extensionPoint
	addTo           map[string][]map[string]string
}

func readFeatureFor(fsys fs.FS, featureName, starterName string) (feature, bool, error) {
	root := path.Join("features", featureName)
	definitionPath := path.Join(root, "feature.yml")
	definitionYAML, err := fs.ReadFile(fsys, definitionPath)
	if err != nil {
		return feature{}, false, fmt.Errorf("read %s: %w", definitionPath, err)
	}
	var definition struct {
		Targets []string `yaml:"targets"`
		Files   []struct {
			Path string `yaml:"path"`
		} `yaml:"files"`
		Delimiters []struct {
			Glob  string `yaml:"glob"`
			Left  string `yaml:"left"`
			Right string `yaml:"right"`
		} `yaml:"delimiters"`
		ExtensionPoints map[string]struct {
			Summary string   `yaml:"summary"`
			File    string   `yaml:"file"`
			Item    []string `yaml:"item"`
		} `yaml:"extension_points"`
		AddTo   map[string][]map[string]string `yaml:"add_to"`
		Options map[string]struct {
			Default string   `yaml:"default"`
			Pattern string   `yaml:"pattern"`
			Values  []string `yaml:"values"`
		} `yaml:"options"`
	}
	if err := yaml.Unmarshal(definitionYAML, &definition); err != nil {
		return feature{}, false, fmt.Errorf("read %s: %w", definitionPath, err)
	}
	if !slices.Contains(definition.Targets, starterName) {
		return feature{}, false, nil
	}
	var rules []delimitersRule
	for _, declared := range definition.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return feature{}, false, fmt.Errorf("read %s: %w", definitionPath, err)
		}
		rules = append(rules, rule)
	}
	var templates []template
	for _, entry := range definition.Files {
		entryRoot := path.Join(root, entry.Path)
		if !strings.HasPrefix(entryRoot, root+"/") {
			return feature{}, false, fmt.Errorf("read %s: the files entry %q is outside the folder of the feature", definitionPath, entry.Path)
		}
		read, err := readTemplates(fsys, entryRoot, featureName)
		if err != nil {
			return feature{}, false, err
		}
		templates = append(templates, read...)
	}
	declaredOptions := options{declarer: "the feature " + featureName, declared: make(map[string]option, len(definition.Options))}
	for optionName, declared := range definition.Options {
		declaredOptions.declared[optionName], err = newOption(optionName, declared.Default, declared.Pattern, declared.Values)
		if err != nil {
			return feature{}, false, fmt.Errorf("read %s: %w", definitionPath, err)
		}
	}
	extensionPoints := make(map[string]extensionPoint, len(definition.ExtensionPoints))
	for name, declared := range definition.ExtensionPoints {
		extensionPoints[name] = extensionPoint{name: name, declarer: "the feature " + featureName, itemFields: declared.Item}
	}
	return feature{name: featureName, templates: templates, rules: rules, options: declaredOptions, extensionPoints: extensionPoints, addTo: definition.AddTo}, true, nil
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
