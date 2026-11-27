package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/urfave/cli/v3"
)

const catalogFlagName = "catalog"

type catalogsLoader func(ctx context.Context, name string) (catalog.Catalogs, error)

type folderLoader func(name string) (string, error)

func catalogFlag(usage string) cli.Flag {
	return &cli.StringFlag{Name: catalogFlagName, Usage: usage}
}

func catalogsToRead(ctx context.Context, cmd *cli.Command, load catalogsLoader) (catalog.Catalogs, error) {
	given := cmd.String(catalogFlagName)
	isFolder, err := givenFolder(given)
	if err != nil {
		return catalog.Catalogs{}, err
	}
	if isFolder {
		return catalog.NewCatalogs(catalog.New(os.DirFS(given))), nil
	}
	return load(ctx, given)
}

func catalogDirOf(cmd *cli.Command, configured folderLoader) (string, error) {
	given := cmd.String(catalogFlagName)
	isFolder, err := givenFolder(given)
	if err != nil || isFolder {
		return given, err
	}
	return configured(given)
}

func givenFolder(given string) (bool, error) {
	if given == "" {
		return false, nil
	}
	info, err := os.Stat(given)
	switch {
	case err == nil && info.IsDir():
		return true, nil
	case errors.Is(err, fs.ErrNotExist) && config.CheckCatalogName(given) == nil:
		return false, nil
	}
	return false, fmt.Errorf("the catalog folder %s is not a folder", given)
}
