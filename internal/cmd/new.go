package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hpcsc/feature-compose/catalog"
	"github.com/urfave/cli/v3"
)

func newNewCommand(c catalog.Catalog) *cli.Command {
	return &cli.Command{
		Name:      "new",
		Usage:     "make a new repo from a starter",
		ArgsUsage: "<starter> <name>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "option", Usage: "set an option of the starter, as <name>=<value>; give the flag one time for each option"},
			&cli.StringFlag{Name: "output-dir", Usage: "the folder for the new repo (default: ./<name>)"},
			&cli.BoolFlag{Name: "no-prompt", Usage: "ask no question, and give each option without a value its default"},
		},
		DisableSliceFlagSeparator: true,
		Action: asTwoArgumentsAction(func(ctx context.Context, cmd *cli.Command, starterName, name string) error {
			values, err := optionValues(cmd.StringSlice("option"))
			if err != nil {
				return err
			}
			starter, err := c.Starter(starterName)
			if err != nil {
				return err
			}
			selection, err := starter.Select(name, values, nil)
			if err != nil {
				return err
			}
			output := cmd.String("output-dir")
			if output == "" {
				output = name
			}
			if err := selection.Write(output); err != nil {
				return err
			}
			if err := selection.RunAfterRender(ctx, output); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.Root().Writer, "Made %s from the starter %s.\n", output, starterName)
			return err
		}, "new needs two arguments: new <starter> <name>"),
	}
}

func optionValues(flags []string) (map[string]string, error) {
	values := make(map[string]string, len(flags))
	for _, flag := range flags {
		name, value, ok := strings.Cut(flag, "=")
		if !ok {
			return nil, fmt.Errorf("the option %q has no value: give it as <name>=<value>", flag)
		}
		values[name] = value
	}
	return values, nil
}
