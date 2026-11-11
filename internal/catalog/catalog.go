package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
)

type Catalog struct {
	fsys fs.FS
}

func New(fsys fs.FS) Catalog {
	return Catalog{fsys: fsys}
}

func (c Catalog) Starter(starterName string) (Starter, error) {
	names, err := c.starterNames()
	if err != nil {
		return Starter{}, err
	}
	if !slices.Contains(names, starterName) {
		return Starter{}, unknownStarter(starterName, names)
	}
	features, err := c.featuresTargeting(starterName)
	if err != nil {
		return Starter{}, err
	}
	return readStarter(c.fsys, starterName, features)
}

func (c Catalog) Combinations() ([]Combination, error) {
	names, err := c.starterNames()
	if err != nil {
		return nil, err
	}
	var combinations []Combination
	for _, starterName := range names {
		starter, err := c.Starter(starterName)
		if err != nil {
			return nil, err
		}
		combinations = append(combinations, starter.combinations()...)
	}
	return combinations, nil
}

func (c Catalog) Definitions() (Definitions, error) {
	starterNames, err := c.starterNames()
	if err != nil {
		return Definitions{}, err
	}
	featureNames, err := c.featureNames()
	if err != nil {
		return Definitions{}, err
	}
	definitions := Definitions{Starters: make(map[string]StarterDefinition, len(starterNames)), Features: make(map[string]FeatureDefinition, len(featureNames))}
	for _, starterName := range starterNames {
		parsed, err := readStarterYML(c.fsys, starterName)
		if err != nil {
			return Definitions{}, err
		}
		definitions.Starters[starterName] = parsed.StarterDefinition.filled()
	}
	for _, featureName := range featureNames {
		parsed, err := readFeatureYML(c.fsys, featureName)
		if err != nil {
			return Definitions{}, err
		}
		definitions.Features[featureName] = parsed.FeatureDefinition.filled()
	}
	return definitions, nil
}

func (c Catalog) Check() ([]string, error) {
	names, err := c.starterNames()
	if err != nil {
		return nil, err
	}
	var mistakes []string
	var starters []Starter
	for _, starterName := range names {
		starter, err := c.Starter(starterName)
		if err != nil {
			return nil, err
		}
		starters = append(starters, starter)
		mistakes = append(mistakes, starter.mistakes()...)
	}
	return append(mistakes, c.itemFieldsMistakes(starters)...), nil
}

func (c Catalog) itemFieldsMistakes(starters []Starter) []string {
	var featureNames []string
	for _, s := range starters {
		featureNames = slices.AppendSeq(featureNames, maps.Keys(s.features))
	}
	slices.Sort(featureNames)
	var mistakes []string
	for _, featureName := range slices.Compact(featureNames) {
		var targets []Starter
		for _, s := range starters {
			if _, ok := s.features[featureName]; ok {
				targets = append(targets, s)
			}
		}
		for _, name := range slices.Sorted(maps.Keys(targets[0].features[featureName].addTo)) {
			var declarations []string
			var first extensionPoint
			differ := false
			for _, s := range targets {
				declared, ok := s.extensionPointsDeclaredFor(s.features[featureName])[name]
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
				mistakes = append(mistakes, fmt.Sprintf("the feature %s adds to the extension point %q, which its targets declare with different item fields: %s", featureName, name, strings.Join(declarations, ", ")))
			}
		}
	}
	return mistakes
}

func (c Catalog) featuresTargeting(starterName string) (map[string]feature, error) {
	featureNames, err := c.featureNames()
	if err != nil {
		return nil, err
	}
	features := map[string]feature{}
	for _, featureName := range featureNames {
		f, targets, err := readFeatureFor(c.fsys, featureName, starterName)
		if err != nil {
			return nil, err
		}
		if targets {
			features[f.name] = f
		}
	}
	return features, nil
}

func (c Catalog) starterNames() ([]string, error) {
	entries, err := fs.ReadDir(c.fsys, "starters")
	if err != nil {
		return nil, fmt.Errorf("read the starters of the catalog: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func (c Catalog) featureNames() ([]string, error) {
	entries, err := fs.ReadDir(c.fsys, "features")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the features of the catalog: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}
