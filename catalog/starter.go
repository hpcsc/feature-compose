package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type Starter struct {
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
		Folders map[string]struct {
			Path string `yaml:"path"`
		} `yaml:"folders"`
	}
	if err := yaml.Unmarshal(definitionYAML, &definition); err != nil {
		return Starter{}, fmt.Errorf("read %s: %w", definitionPath, err)
	}
	folders := make(map[string]folder, len(definition.Folders))
	for folderName, declared := range definition.Folders {
		if !filepath.IsLocal(filepath.FromSlash(declared.Path)) {
			return Starter{}, fmt.Errorf("read %s: the path %q of the folder %s is outside the repo", definitionPath, declared.Path, folderName)
		}
		folders[folderName] = folder{path: declared.Path}
	}
	templates, err := readTemplates(fsys, path.Join(root, "files"), folders)
	if err != nil {
		return Starter{}, err
	}
	return Starter{templates: templates}, nil
}

func (s Starter) Select(name string) Selection {
	return Selection{starter: s, name: name}
}
