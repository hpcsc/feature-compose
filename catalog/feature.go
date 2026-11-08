package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

type feature struct {
	name      string
	templates []template
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
	}
	if err := yaml.Unmarshal(definitionYAML, &definition); err != nil {
		return feature{}, false, fmt.Errorf("read %s: %w", definitionPath, err)
	}
	if !slices.Contains(definition.Targets, starterName) {
		return feature{}, false, nil
	}
	var templates []template
	for _, entry := range definition.Files {
		entryRoot := path.Join(root, entry.Path)
		if !strings.HasPrefix(entryRoot, root+"/") {
			return feature{}, false, fmt.Errorf("read %s: the files entry %q is outside the folder of the feature", definitionPath, entry.Path)
		}
		read, err := readTemplates(fsys, entryRoot, "the feature "+featureName)
		if err != nil {
			return feature{}, false, err
		}
		templates = append(templates, read...)
	}
	return feature{name: featureName, templates: templates}, true, nil
}
