package catalog

import (
	"fmt"
	"io/fs"
	"path"

	"go.yaml.in/yaml/v3"
)

type featureYML struct {
	FeatureDefinition `yaml:",inline"`
	NeedsNode         yaml.Node `yaml:"needs"`
	Files             []struct {
		Path string `yaml:"path"`
	} `yaml:"files"`
	Delimiters []struct {
		Glob  string `yaml:"glob"`
		Left  string `yaml:"left"`
		Right string `yaml:"right"`
	} `yaml:"delimiters"`
	needNames []string
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
	parsed, err = parsed.withNeeds()
	if err != nil {
		return featureYML{}, fmt.Errorf("read %s: %w", ymlPath, err)
	}
	return parsed, nil
}

func (y featureYML) withNeeds() (featureYML, error) {
	y.Needs = map[string]map[string]string{}
	switch {
	case y.NeedsNode.Kind == 0 || y.NeedsNode.Tag == "!!null":
		return y, nil
	case y.NeedsNode.Kind == yaml.SequenceNode:
		for _, entry := range y.NeedsNode.Content {
			if entry.Kind != yaml.ScalarNode {
				return y, y.needsOfAnotherForm()
			}
			if _, ok := y.Needs[entry.Value]; ok {
				return y, y.needNamedTwoTimes(entry.Value)
			}
			y.needNames = append(y.needNames, entry.Value)
			y.Needs[entry.Value] = map[string]string{}
		}
		return y, nil
	case y.NeedsNode.Kind == yaml.MappingNode:
		for i := 0; i < len(y.NeedsNode.Content); i += 2 {
			name, values := y.NeedsNode.Content[i], y.NeedsNode.Content[i+1]
			var decoded map[string]string
			if err := values.Decode(&decoded); err != nil {
				return y, fmt.Errorf("needs: the values of the need %s must be a map from each option to its value: %w", name.Value, err)
			}
			if decoded == nil {
				decoded = map[string]string{}
			}
			if _, ok := y.Needs[name.Value]; ok {
				return y, y.needNamedTwoTimes(name.Value)
			}
			y.needNames = append(y.needNames, name.Value)
			y.Needs[name.Value] = decoded
		}
		return y, nil
	}
	return y, y.needsOfAnotherForm()
}

func (y featureYML) needNamedTwoTimes(needName string) error {
	return fmt.Errorf("needs names the feature %s two times", needName)
}

func (y featureYML) needsOfAnotherForm() error {
	return fmt.Errorf("needs at line %d must be a list of feature names, or a map from each feature name to the values that its need sets", y.NeedsNode.Line)
}
