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

func catalogToRead(cmd *cli.Command, load func() (catalog.Catalogs, error)) (catalog.Catalogs, error) {
	catalogDir := cmd.String(catalogFlagName)
	if catalogDir == "" {
		return load()
	}
	if err := checkCatalogFolder(catalogDir); err != nil {
		return catalog.Catalogs{}, err
	}
	return catalog.NewCatalogs(catalog.New(os.DirFS(catalogDir))), nil
}

func catalogDirOf(cmd *cli.Command, configured func() (string, error)) (string, error) {
	catalogDir := cmd.String(catalogFlagName)
	if catalogDir == "" {
		return configured()
	}
	return catalogDir, checkCatalogFolder(catalogDir)
}

func checkCatalogFolder(catalogDir string) error {
	info, err := os.Stat(catalogDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("the catalog folder %s is not a folder", catalogDir)
	}
	return nil
}
