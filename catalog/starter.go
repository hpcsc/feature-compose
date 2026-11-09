package catalog

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

var repoName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Starter struct {
	name      string
	options   map[string]option
	folders   map[string]folder
	templates []template
	features  map[string]feature
}

func readStarter(fsys fs.FS, starterName string, features map[string]feature) (Starter, error) {
	root := path.Join("starters", starterName)
	definitionPath := path.Join(root, "starter.yml")
	definitionYAML, err := fs.ReadFile(fsys, definitionPath)
	if err != nil {
		return Starter{}, fmt.Errorf("read %s: %w", definitionPath, err)
	}
	var definition struct {
		Options map[string]struct {
			Default string `yaml:"default"`
			Pattern string `yaml:"pattern"`
		} `yaml:"options"`
		Folders map[string]struct {
			Path        string   `yaml:"path"`
			AfterRender []string `yaml:"after_render"`
		} `yaml:"folders"`
		Delimiters []struct {
			Glob  string `yaml:"glob"`
			Left  string `yaml:"left"`
			Right string `yaml:"right"`
		} `yaml:"delimiters"`
	}
	if err := yaml.Unmarshal(definitionYAML, &definition); err != nil {
		return Starter{}, fmt.Errorf("read %s: %w", definitionPath, err)
	}
	options := make(map[string]option, len(definition.Options))
	for optionName, declared := range definition.Options {
		declaredOption := option{name: optionName, defaultValue: declared.Default}
		if declared.Pattern != "" {
			declaredOption.pattern, err = regexp.Compile(declared.Pattern)
			if err != nil {
				return Starter{}, fmt.Errorf("read %s: the pattern of the option %s: %w", definitionPath, optionName, err)
			}
		}
		options[optionName] = declaredOption
	}
	folders := make(map[string]folder, len(definition.Folders))
	for folderName, declared := range definition.Folders {
		if !filepath.IsLocal(filepath.FromSlash(declared.Path)) {
			return Starter{}, fmt.Errorf("read %s: the path %q of the folder %s is outside the repo", definitionPath, declared.Path, folderName)
		}
		folders[folderName] = folder{name: folderName, path: declared.Path, afterRender: declared.AfterRender}
	}
	var rules []delimitersRule
	for _, declared := range definition.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return Starter{}, fmt.Errorf("read %s: %w", definitionPath, err)
		}
		rules = append(rules, rule)
	}
	read, err := readTemplates(fsys, path.Join(root, "files"), "the starter")
	if err != nil {
		return Starter{}, err
	}
	templates := make([]template, 0, len(read))
	for _, t := range read {
		placed, err := t.place(folders, rules)
		if err != nil {
			return Starter{}, err
		}
		templates = append(templates, placed)
	}
	return Starter{name: starterName, options: options, folders: folders, templates: templates, features: features}, nil
}

func (s Starter) Select(name string, given map[string]string, featureNames []string) (Selection, error) {
	if !repoName.MatchString(name) {
		return Selection{}, fmt.Errorf("the name %q of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'", name)
	}
	for _, optionName := range slices.Sorted(maps.Keys(given)) {
		declared, ok := s.options[optionName]
		if !ok {
			return Selection{}, fmt.Errorf("unknown option %q: the options of the starter are %s",
				optionName, strings.Join(slices.Sorted(maps.Keys(s.options)), ", "))
		}
		if err := declared.check(given[optionName]); err != nil {
			return Selection{}, err
		}
	}
	selected, err := s.selectedFeatures(featureNames)
	if err != nil {
		return Selection{}, err
	}
	templates, err := s.templatesOf(selected)
	if err != nil {
		return Selection{}, err
	}
	extensionPoints, err := s.extensionPointsOf(selected)
	if err != nil {
		return Selection{}, err
	}
	selection := Selection{folders: s.folders, name: name, templates: templates, extensionPoints: extensionPoints}
	values := make(map[string]string, len(s.options))
	for optionName, declared := range s.options {
		value, ok := given[optionName]
		if !ok {
			var err error
			value, err = selection.render("the default of the option "+optionName, declared.defaultValue, defaultDelimiters)
			if err != nil {
				return Selection{}, err
			}
		}
		values[optionName] = value
	}
	selection.values = values
	return selection, nil
}

func (s Starter) selectedFeatures(featureNames []string) ([]feature, error) {
	byName := map[string]feature{}
	for _, featureName := range featureNames {
		f, ok := s.features[featureName]
		if !ok {
			return nil, s.unknownFeature(featureName)
		}
		byName[featureName] = f
	}
	selected := make([]feature, 0, len(byName))
	for _, featureName := range slices.Sorted(maps.Keys(byName)) {
		selected = append(selected, byName[featureName])
	}
	return selected, nil
}

func (s Starter) templatesOf(selected []feature) ([]template, error) {
	templates := slices.Clone(s.templates)
	for _, f := range selected {
		placed, err := f.placedIn(s.folders)
		if err != nil {
			return nil, err
		}
		templates = append(templates, placed...)
	}
	return templates, nil
}

func (s Starter) extensionPointsOf(selected []feature) ([]string, error) {
	declarers := map[string]string{}
	for _, f := range selected {
		for _, extensionPoint := range f.extensionPoints {
			if other, ok := declarers[extensionPoint]; ok {
				return nil, fmt.Errorf("the features %s and %s both declare the extension point %q", other, f.name, extensionPoint)
			}
			declarers[extensionPoint] = f.name
		}
	}
	return slices.Sorted(maps.Keys(declarers)), nil
}

func (s Starter) unknownFeature(featureName string) error {
	if len(s.features) == 0 {
		return fmt.Errorf("unknown feature %q: no feature targets the starter %s", featureName, s.name)
	}
	return fmt.Errorf("unknown feature %q: the features that target the starter %s are %s",
		featureName, s.name, strings.Join(slices.Sorted(maps.Keys(s.features)), ", "))
}
