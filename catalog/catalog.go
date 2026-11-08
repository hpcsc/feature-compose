package catalog

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

//go:embed all:starters
var starters embed.FS

type Catalog struct {
	fsys fs.FS
}

func New(fsys fs.FS) Catalog {
	return Catalog{fsys: fsys}
}

func Embedded() Catalog {
	return New(starters)
}

func (c Catalog) Starter(starterName string) (Starter, error) {
	names, err := c.starterNames()
	if err != nil {
		return Starter{}, err
	}
	if !slices.Contains(names, starterName) {
		return Starter{}, fmt.Errorf("unknown starter %q: the catalog has %s", starterName, strings.Join(names, ", "))
	}
	features, err := c.featuresTargeting(starterName)
	if err != nil {
		return Starter{}, err
	}
	return readStarter(c.fsys, starterName, features)
}

func (c Catalog) featuresTargeting(starterName string) (map[string]feature, error) {
	entries, err := fs.ReadDir(c.fsys, "features")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the features of the catalog: %w", err)
	}
	features := map[string]feature{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		f, targets, err := readFeatureFor(c.fsys, entry.Name(), starterName)
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
