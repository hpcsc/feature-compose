package cmd

import (
	"fmt"
	"os"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/urfave/cli/v3"
)

const catalogFlagName = "catalog"

func catalogFlag(usage string) cli.Flag {
	return &cli.StringFlag{Name: catalogFlagName, Usage: usage}
}

func catalogToRead(cmd *cli.Command, load func() (catalog.Catalog, error)) (catalog.Catalog, error) {
	catalogDir := cmd.String(catalogFlagName)
	if catalogDir == "" {
		return load()
	}
	info, err := os.Stat(catalogDir)
	if err != nil || !info.IsDir() {
		return catalog.Catalog{}, fmt.Errorf("the catalog folder %s is not a folder", catalogDir)
	}
	return catalog.New(os.DirFS(catalogDir)), nil
}
