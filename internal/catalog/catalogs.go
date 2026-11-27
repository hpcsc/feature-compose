package catalog

import (
	"context"
	"fmt"
	"maps"
	"slices"
)

type Catalogs struct {
	all []Catalog
}

func NewCatalogs(all ...Catalog) Catalogs {
	return Catalogs{all: all}
}

func (cs Catalogs) named(name string) (Catalog, bool) {
	for _, c := range cs.all {
		if c.origin.Name == name {
			return c, true
		}
	}
	return Catalog{}, false
}

func (cs Catalogs) Starter(name string) (Starter, error) {
	byFullName := map[string]Catalog{}
	for _, c := range cs.all {
		starterNames, err := c.starterNames()
		if err != nil {
			return Starter{}, cs.inCatalog(c, err)
		}
		for _, starterName := range starterNames {
			byFullName[c.origin.fullName(starterName)] = c
		}
	}
	fullName, err := fullNames(slices.Sorted(maps.Keys(byFullName))).find(name)
	if err != nil {
		return Starter{}, err
	}
	c := byFullName[fullName]
	_, starterName := SplitFullName(fullName)
	starter, err := c.Starter(starterName)
	return starter, cs.inCatalog(c, err)
}

func (cs Catalogs) Definitions() (Definitions, error) {
	all := Definitions{Starters: map[string]StarterDefinition{}, Features: map[string]FeatureDefinition{}}
	for _, c := range cs.all {
		definitions, err := c.Definitions()
		if err != nil {
			return Definitions{}, cs.inCatalog(c, err)
		}
		maps.Copy(all.Starters, definitions.Starters)
		maps.Copy(all.Features, definitions.Features)
		if len(definitions.Catalogs) > 0 {
			if all.Catalogs == nil {
				all.Catalogs = map[string]Origin{}
			}
			maps.Copy(all.Catalogs, definitions.Catalogs)
		}
	}
	return all, nil
}

func (cs Catalogs) Check(ctx context.Context) ([]string, error) {
	var all []string
	for _, c := range cs.all {
		mistakes, err := c.Check(ctx)
		if err != nil {
			return nil, cs.inCatalog(c, err)
		}
		for _, mistake := range mistakes {
			if c.origin.Name != "" {
				mistake = "the catalog " + c.origin.Name + ": " + mistake
			}
			all = append(all, mistake)
		}
	}
	return all, nil
}

func (cs Catalogs) Combinations() ([]Combination, error) {
	var all []Combination
	for _, c := range cs.all {
		combinations, err := c.Combinations()
		if err != nil {
			return nil, cs.inCatalog(c, err)
		}
		all = append(all, combinations...)
	}
	return all, nil
}

func (cs Catalogs) TestCombination(ctx context.Context, combination Combination, name, output string) error {
	c, ok := cs.named(combination.CatalogName)
	if !ok {
		return fmt.Errorf("no catalog has the name %q", combination.CatalogName)
	}
	return c.TestCombination(ctx, combination, name, output)
}

func (cs Catalogs) inCatalog(c Catalog, err error) error {
	if err == nil || c.origin.Name == "" {
		return err
	}
	return fmt.Errorf("the catalog %s: %w", c.origin.Name, err)
}
