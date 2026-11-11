package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/urfave/cli/v3"
)

func newCheckCommand(load func() (catalog.Catalog, error)) *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "find each feature of the catalog that does not fit its targets or its needs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "catalog", Usage: "check the catalog in this folder, not the catalog folder of the config file"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return errors.New("check takes no argument")
			}
			c, err := catalogToCheck(cmd.String("catalog"), load)
			if err != nil {
				return err
			}
			mistakes, err := c.Check()
			if err != nil {
				return err
			}
			if len(mistakes) == 0 {
				_, err = fmt.Fprintln(cmd.Root().Writer, "The catalog has no mistakes.")
				return err
			}
			if _, err := fmt.Fprintln(cmd.Root().Writer, strings.Join(mistakes, "\n")); err != nil {
				return err
			}
			if len(mistakes) == 1 {
				return errors.New("the catalog has 1 mistake")
			}
			return fmt.Errorf("the catalog has %d mistakes", len(mistakes))
		},
	}
}

func catalogToCheck(catalogDir string, load func() (catalog.Catalog, error)) (catalog.Catalog, error) {
	if catalogDir == "" {
		return load()
	}
	info, err := os.Stat(catalogDir)
	if err != nil || !info.IsDir() {
		return catalog.Catalog{}, fmt.Errorf("the catalog folder %s is not a folder", catalogDir)
	}
	return catalog.New(os.DirFS(catalogDir)), nil
}
