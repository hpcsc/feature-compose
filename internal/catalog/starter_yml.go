package catalog

import (
	"fmt"
	"io/fs"
	"path"

	"go.yaml.in/yaml/v3"
)

type starterYML struct {
	StarterDefinition `yaml:",inline"`
	Delimiters        []struct {
		Glob  string `yaml:"glob"`
		Left  string `yaml:"left"`
		Right string `yaml:"right"`
	} `yaml:"delimiters"`
}

func starterYMLPath(starterName string) string {
	return path.Join("starters", starterName, "starter.yml")
}

func readStarterYML(fsys fs.FS, starterName string) (starterYML, error) {
	ymlPath := starterYMLPath(starterName)
	contents, err := fs.ReadFile(fsys, ymlPath)
	if err != nil {
		return starterYML{}, fmt.Errorf("read %s: %w", ymlPath, err)
	}
	var parsed starterYML
	if err := yaml.Unmarshal(contents, &parsed); err != nil {
		return starterYML{}, fmt.Errorf("read %s: %w", ymlPath, err)
	}
	return parsed, nil
}
