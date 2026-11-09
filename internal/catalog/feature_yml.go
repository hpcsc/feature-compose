package catalog

import (
	"fmt"
	"io/fs"
	"path"

	"go.yaml.in/yaml/v3"
)

type featureYML struct {
	FeatureDefinition `yaml:",inline"`
	Files             []struct {
		Path string `yaml:"path"`
	} `yaml:"files"`
	Delimiters []struct {
		Glob  string `yaml:"glob"`
		Left  string `yaml:"left"`
		Right string `yaml:"right"`
	} `yaml:"delimiters"`
}

func featureYMLPath(featureName string) string {
	return path.Join("features", featureName, "feature.yml")
}

func readFeatureYML(fsys fs.FS, featureName string) (featureYML, error) {
	ymlPath := featureYMLPath(featureName)
	contents, err := fs.ReadFile(fsys, ymlPath)
	if err != nil {
		return featureYML{}, fmt.Errorf("read %s: %w", ymlPath, err)
	}
	var parsed featureYML
	if err := yaml.Unmarshal(contents, &parsed); err != nil {
		return featureYML{}, fmt.Errorf("read %s: %w", ymlPath, err)
	}
	return parsed, nil
}
