package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
)

type Catalog struct {
	fsys   fs.FS
	origin Origin
}

func New(fsys fs.FS) Catalog {
	return Catalog{fsys: fsys}
}

func FromOrigin(fsys fs.FS, origin Origin) Catalog {
	return Catalog{fsys: fsys, origin: origin}
}

func (c Catalog) Starter(starterName string) (Starter, error) {
	names, err := c.starterNames()
	if err != nil {
		return Starter{}, err
	}
	if !slices.Contains(names, starterName) {
		return Starter{}, fullNames(names).unknown(starterName)
	}
	features, otherFeatureNames, err := c.featuresFor(starterName)
	if err != nil {
		return Starter{}, err
	}
	starter, err := readStarter(c.fsys, starterName, features, otherFeatureNames)
	starter.origin = c.origin
	return starter, err
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
		definitions.Starters[c.origin.fullName(starterName)] = parsed.StarterDefinition.filled()
	}
	for _, featureName := range featureNames {
		parsed, err := readFeatureYML(c.fsys, featureName)
		if err != nil {
			return Definitions{}, err
		}
		definitions.Features[c.origin.fullName(featureName)] = parsed.FeatureDefinition.filled().withFullNames(c.origin)
	}
	return definitions, nil
}

func (c Catalog) Check(ctx context.Context) ([]string, error) {
	names, err := c.starterNames()
	if err != nil {
		return nil, err
	}
	catalogFeatures, err := c.features()
	if err != nil {
		return nil, err
	}
	var mistakes []string
	var starters []Starter
	for _, starterName := range names {
		targeting := maps.Clone(catalogFeatures)
		var otherFeatureNames []string
		maps.DeleteFunc(targeting, func(featureName string, f feature) bool {
			if f.targetsStarter(starterName) {
				return false
			}
			otherFeatureNames = append(otherFeatureNames, featureName)
			return true
		})
		starter, err := readStarter(c.fsys, starterName, targeting, otherFeatureNames)
		if err != nil {
			return nil, err
		}
		starters = append(starters, starter)
		mistakes = append(mistakes, starter.mistakes(ctx, catalogFeatures)...)
	}
	for _, featureName := range slices.Sorted(maps.Keys(catalogFeatures)) {
		mistakes = append(mistakes, catalogFeatures[featureName].mistakes(starters, catalogFeatures)...)
	}
	return mistakes, nil
}

func (c Catalog) features() (map[string]feature, error) {
	featureNames, err := c.featureNames()
	if err != nil {
		return nil, err
	}
	features := map[string]feature{}
	for _, featureName := range featureNames {
		f, err := readFeature(c.fsys, featureName)
		if err != nil {
			return nil, err
		}
		features[f.name] = f
	}
	return features, nil
}

func (c Catalog) featuresFor(starterName string) (map[string]feature, []string, error) {
	featureNames, err := c.featureNames()
	if err != nil {
		return nil, nil, err
	}
	features := map[string]feature{}
	var otherFeatureNames []string
	for _, featureName := range featureNames {
		f, targets, err := readFeatureFor(c.fsys, featureName, starterName)
		if err != nil {
			return nil, nil, err
		}
		if targets {
			features[f.name] = f
		} else {
			otherFeatureNames = append(otherFeatureNames, featureName)
		}
	}
	return features, otherFeatureNames, nil
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
