package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/urfave/cli/v3"
)

func newCheckCommand(load func() (catalog.Catalog, error)) *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "find each feature of the catalog that does not fit its targets or its needs",
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return errors.New("check takes no argument")
			}
			c, err := load()
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
