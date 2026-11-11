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
	targets         []string
	templates       []template
	rules           []delimitersRule
	options         options
	extensionPoints map[string]extensionPoint
	addTo           map[string][]map[string]string
	needs           []string
	needValues      map[string]map[string]string
}

func readFeature(fsys fs.FS, featureName string) (feature, error) {
	parsed, err := readFeatureYML(fsys, featureName)
	if err != nil {
		return feature{}, err
	}
	return newFeature(fsys, featureName, parsed)
}

func readFeatureFor(fsys fs.FS, featureName, starterName string) (feature, bool, error) {
	parsed, err := readFeatureYML(fsys, featureName)
	if err != nil {
		return feature{}, false, err
	}
	if !slices.Contains(parsed.Targets, starterName) {
		return feature{}, false, nil
	}
	f, err := newFeature(fsys, featureName, parsed)
	if err != nil {
		return feature{}, false, err
	}
	return f, true, nil
}

func newFeature(fsys fs.FS, featureName string, parsed featureYML) (feature, error) {
	root := path.Join("features", featureName)
	ymlPath := featureYMLPath(featureName)
	var rules []delimitersRule
	for _, declared := range parsed.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return feature{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		rules = append(rules, rule)
	}
	var templates []template
	for _, entry := range parsed.Files {
		entryRoot := path.Join(root, entry.Path)
		if !strings.HasPrefix(entryRoot, root+"/") {
			return feature{}, fmt.Errorf("read %s: the files entry %q is outside the folder of the feature", ymlPath, entry.Path)
		}
		read, err := readTemplates(fsys, entryRoot, featureName)
		if err != nil {
			return feature{}, err
		}
		var when expression
		if entry.When != "" {
			when, err = newExpression(entry.When)
			if err != nil {
				return feature{}, fmt.Errorf("read %s: the when %q of the files entry %q: %w", ymlPath, entry.When, entry.Path, err)
			}
		}
		for i := range read {
			read[i].entry, read[i].when = entry.Path, when
		}
		templates = append(templates, read...)
	}
	declaredOptions := options{declarer: "the feature " + featureName, declared: make(map[string]option, len(parsed.Options))}
	for optionName, declared := range parsed.Options {
		var err error
		declaredOptions.declared[optionName], err = newOption(optionName, declared.Default, declared.Pattern, declared.Values)
		if err != nil {
			return feature{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
	}
	extensionPoints := make(map[string]extensionPoint, len(parsed.ExtensionPoints))
	for name, declared := range parsed.ExtensionPoints {
		extensionPoints[name] = extensionPoint{name: name, declarer: "the feature " + featureName, itemFields: declared.Item}
	}
	return feature{
		name: featureName, targets: parsed.Targets, templates: templates, rules: rules, options: declaredOptions,
		extensionPoints: extensionPoints, addTo: parsed.AddTo, needs: parsed.needNames, needValues: parsed.Needs,
	}, nil
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

func (f feature) targetsStarter(starterName string) bool {
	return slices.Contains(f.targets, starterName)
}

func (f feature) needMistakes(starterName string, catalogFeatures map[string]feature) []string {
	var mistakes []string
	for _, needName := range f.needs {
		needed, ok := catalogFeatures[needName]
		switch {
		case !ok:
			mistakes = append(mistakes, fmt.Sprintf("the feature %s needs %s, which is not a feature of the catalog", f.name, needName))
		case !needed.targetsStarter(starterName):
			mistakes = append(mistakes, fmt.Sprintf("the feature %s needs %s, which does not target the starter", f.name, needName))
		}
	}
	for _, needName := range slices.Sorted(maps.Keys(f.needValues)) {
		needed, ok := catalogFeatures[needName]
		if !ok {
			continue
		}
		values := f.needValues[needName]
		for _, optionName := range slices.Sorted(maps.Keys(values)) {
			if err := needed.options.check(map[string]string{optionName: values[optionName]}); err != nil {
				mistakes = append(mistakes, fmt.Sprintf("the need of the feature %s: %s", f.name, err))
			}
		}
	}
	return mistakes
}

func (f feature) itemFieldsMistakes(starters []Starter) []string {
	declaredByTarget := map[string]map[string]extensionPoint{}
	for _, s := range starters {
		if f.targetsStarter(s.name) {
			declaredByTarget[s.name] = s.extensionPointsDeclaredFor(f)
		}
	}
	var mistakes []string
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		var declarations []string
		var first extensionPoint
		differ := false
		for _, s := range starters {
			declared, ok := declaredByTarget[s.name][name]
			if !ok {
				continue
			}
			if declarations == nil {
				first = declared
			}
			differ = differ || !declared.hasItemFields(first.itemFields)
			declarations = append(declarations, fmt.Sprintf("[%s] in the starter %s", strings.Join(declared.itemFields, ", "), s.name))
		}
		if differ {
			mistakes = append(mistakes, fmt.Sprintf("the feature %s adds to the extension point %q, which its targets declare with different item fields: %s", f.name, name, strings.Join(declarations, ", ")))
		}
	}
	return mistakes
}

func (f feature) addsToOneExtensionPointWith(other feature) bool {
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		if _, ok := other.addTo[name]; ok {
			return true
		}
	}
	return false
}
