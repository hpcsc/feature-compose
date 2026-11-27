package catalog

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	selectionFilePath  = ".feature-compose.yml"
	selectionFileOwner = "feature-compose"
)

type selectionFile struct {
	Version       string                       `yaml:"version"`
	Catalog       *Origin                      `yaml:"catalog,omitempty"`
	StarterName   string                       `yaml:"starter"`
	Name          string                       `yaml:"name"`
	StarterValues map[string]string            `yaml:"options"`
	FeatureValues map[string]map[string]string `yaml:"features"`
	Requested     []string                     `yaml:"requested,flow"`
}

func (f selectionFile) contents() (string, error) {
	var contents strings.Builder
	encoder := yaml.NewEncoder(&contents)
	encoder.SetIndent(2)
	if err := encoder.Encode(f); err != nil {
		return "", fmt.Errorf("write %s: %w", selectionFilePath, err)
	}
	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("write %s: %w", selectionFilePath, err)
	}
	return contents.String(), nil
}
