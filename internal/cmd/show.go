package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/urfave/cli/v3"
)

func newShowCommand(load func() (catalog.Catalog, error)) *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "show the starters, the features and the extension points of the catalog",
		ArgsUsage: "[<starter>]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Usage: "print the catalog in this format: json"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 1 {
				return errors.New("show takes at most one argument: show [<starter>]")
			}
			if format := cmd.String("output"); format != "" && format != "json" {
				return fmt.Errorf("unknown output %q: show prints only json", format)
			}
			c, err := load()
			if err != nil {
				return err
			}
			definitions, err := c.Definitions()
			if err != nil {
				return err
			}
			if starterName := cmd.Args().First(); starterName != "" {
				definitions, err = definitions.ForStarter(starterName)
				if err != nil {
					return err
				}
			}
			encoder := json.NewEncoder(cmd.Root().Writer)
			encoder.SetIndent("", "  ")
			return encoder.Encode(definitions)
		},
	}
}
