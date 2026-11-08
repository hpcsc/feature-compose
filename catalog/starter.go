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

type Starter struct {
	options   map[string]option
	folders   []folder
	templates []template
}

func readStarter(fsys fs.FS, starterName string) (Starter, error) {
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
	templates, err := readTemplates(fsys, path.Join(root, "files"), folders)
	if err != nil {
		return Starter{}, err
	}
	var byName []folder
	for _, folderName := range slices.Sorted(maps.Keys(folders)) {
		byName = append(byName, folders[folderName])
	}
	return Starter{options: options, folders: byName, templates: templates}, nil
}

func (s Starter) Select(name string, given map[string]string) (Selection, error) {
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
	selection := Selection{starter: s, name: name}
	values := make(map[string]string, len(s.options))
	for optionName, declared := range s.options {
		value, ok := given[optionName]
		if !ok {
			var err error
			value, err = selection.render("the default of the option "+optionName, declared.defaultValue)
			if err != nil {
				return Selection{}, err
			}
		}
		values[optionName] = value
	}
	selection.values = values
	return selection, nil
}
