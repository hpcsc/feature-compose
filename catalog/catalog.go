package catalog

import (
	"embed"
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
	return readStarter(c.fsys, starterName)
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
