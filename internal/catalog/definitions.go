package catalog

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/astgrep"
)

type FolderDefinition struct {
	Path        string   `json:"path" yaml:"path"`
	Generate    string   `json:"generate,omitempty" yaml:"generate"`
	AfterRender []string `json:"after_render" yaml:"after_render"`
	Test        []string `json:"test" yaml:"test"`
}

func (f FolderDefinition) filled() FolderDefinition {
	f.AfterRender = nonNil(f.AfterRender)
	f.Test = nonNil(f.Test)
	return f
}

type OptionDefinition struct {
	Default string   `json:"default" yaml:"default"`
	Pattern string   `json:"pattern" yaml:"pattern"`
	Values  []string `json:"values" yaml:"values"`
	When    string   `json:"when" yaml:"when"`
}

func (o OptionDefinition) filled() OptionDefinition {
	o.Values = nonNil(o.Values)
	return o
}

type QueryDefinition struct {
	Language string         `json:"language,omitempty" yaml:"language"`
	Pattern  string         `json:"pattern,omitempty" yaml:"pattern"`
	Rule     map[string]any `json:"rule,omitempty" yaml:"rule"`
}

func (q QueryDefinition) query() astgrep.Query {
	return astgrep.Query{Language: q.Language, Pattern: q.Pattern, Rule: q.Rule}
}

type ExtensionPointDefinition struct {
	Summary         string   `json:"summary" yaml:"summary"`
	File            string   `json:"file" yaml:"file"`
	Item            []string `json:"item" yaml:"item"`
	QueryDefinition `yaml:",inline"`
	After           string `json:"after,omitempty" yaml:"after"`
	Before          string `json:"before,omitempty" yaml:"before"`
	Separator       string `json:"separator,omitempty" yaml:"separator"`
	Insert          string `json:"insert,omitempty" yaml:"insert"`
}

func (e ExtensionPointDefinition) filled() ExtensionPointDefinition {
	e.Item = nonNil(e.Item)
	return e
}

type RemovalDefinition struct {
	File            string `json:"file" yaml:"file"`
	QueryDefinition `yaml:",inline"`
}

type StarterDefinition struct {
	Summary         string                              `json:"summary" yaml:"summary"`
	Folders         map[string]FolderDefinition         `json:"folders" yaml:"folders"`
	Options         map[string]OptionDefinition         `json:"options" yaml:"options"`
	ExtensionPoints map[string]ExtensionPointDefinition `json:"extension_points" yaml:"extension_points"`
	Remove          []RemovalDefinition                 `json:"remove,omitempty" yaml:"remove"`
	AddTo           map[string][]map[string]string      `json:"add_to,omitempty" yaml:"add_to"`
}

func (s StarterDefinition) filled() StarterDefinition {
	s.Folders = filled(s.Folders, FolderDefinition.filled)
	s.Options = filled(s.Options, OptionDefinition.filled)
	s.ExtensionPoints = filled(s.ExtensionPoints, ExtensionPointDefinition.filled)
	return s
}

type FeatureDefinition struct {
	Summary         string                              `json:"summary" yaml:"summary"`
	Targets         []string                            `json:"targets" yaml:"targets"`
	Needs           map[string]map[string]string        `json:"needs" yaml:"-"`
	Options         map[string]OptionDefinition         `json:"options" yaml:"options"`
	ExtensionPoints map[string]ExtensionPointDefinition `json:"extension_points" yaml:"extension_points"`
	AddTo           map[string][]map[string]string      `json:"add_to" yaml:"add_to"`
}

func (f FeatureDefinition) filled() FeatureDefinition {
	f.Targets = nonNil(f.Targets)
	f.Options = filled(f.Options, OptionDefinition.filled)
	f.ExtensionPoints = filled(f.ExtensionPoints, ExtensionPointDefinition.filled)
	f.AddTo = filled(f.AddTo, nonNil[map[string]string])
	return f
}

type Definitions struct {
	Starters map[string]StarterDefinition `json:"starters"`
	Features map[string]FeatureDefinition `json:"features"`
}

func (d Definitions) ForStarter(starterName string) (Definitions, error) {
	starter, ok := d.Starters[starterName]
	if !ok {
		return Definitions{}, unknownStarter(starterName, slices.Sorted(maps.Keys(d.Starters)))
	}
	features := map[string]FeatureDefinition{}
	for featureName, f := range d.Features {
		if slices.Contains(f.Targets, starterName) {
			features[featureName] = f
		}
	}
	return Definitions{Starters: map[string]StarterDefinition{starterName: starter}, Features: features}, nil
}

func unknownStarter(starterName string, starterNames []string) error {
	return fmt.Errorf("unknown starter %q: the catalog has %s", starterName, strings.Join(starterNames, ", "))
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func filled[V any](byName map[string]V, fill func(V) V) map[string]V {
	result := make(map[string]V, len(byName))
	for name, value := range byName {
		result[name] = fill(value)
	}
	return result
}
