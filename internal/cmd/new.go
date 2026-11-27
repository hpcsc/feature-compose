package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/form"
	"github.com/urfave/cli/v3"
)

func newNewCommand(load func() (catalog.Catalog, error), version string) *cli.Command {
	return &cli.Command{
		Name:      "new",
		Usage:     "make a new repo from a starter",
		ArgsUsage: "[<starter> [<name>]]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "feature", Usage: "select a feature of the catalog, as <name>[:<option>=<value>,...]; give the flag one time for each feature"},
			&cli.StringSliceFlag{Name: "option", Usage: "set an option of the starter, as <name>=<value>; give the flag one time for each option"},
			&cli.StringFlag{Name: "output-dir", Usage: "the folder for the new repo (default: ./<name>)"},
			&cli.BoolFlag{Name: "no-prompt", Usage: "start no TUI: give each option without a value its default, and stop on an option with no default and no value"},
			&cli.BoolFlag{Name: "dry-run", Usage: "print a preview of the repo, and write no file and run no command"},
			catalogFlag("make the repo from the catalog in this folder, not the catalog folder of the config file"),
		},
		DisableSliceFlagSeparator: true,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			asked := !cmd.Bool("no-prompt") && !cmd.Bool("dry-run") && isTerminal(cmd.Root().Reader) && isTerminal(cmd.Root().Writer)
			arguments := cmd.Args().Slice()
			if len(arguments) < 2 && !asked {
				return errors.New("new needs two arguments: new <starter> <name>")
			}
			arguments = append(arguments, "", "")
			starterName, name := arguments[0], arguments[1]
			given, err := givenStarterValues(cmd.StringSlice("option"))
			if err != nil {
				return err
			}
			givenByFeature, err := givenFeatureValues(cmd.StringSlice("feature"))
			if err != nil {
				return err
			}
			c, err := catalogToRead(cmd, load)
			if err != nil {
				return err
			}
			var selection catalog.Selection
			if asked {
				selection, err = ask(ctx, cmd, c, starterName, name, given, givenByFeature)
			} else {
				selection, err = selected(c, starterName, name, given, givenByFeature)
			}
			if err != nil {
				return err
			}
			if cmd.Bool("dry-run") {
				preview, err := selection.Preview()
				if err != nil {
					return err
				}
				_, err = io.WriteString(cmd.Root().Writer, preview)
				return err
			}
			output := cmd.String("output-dir")
			if output == "" {
				output = selection.Name()
			}
			steps := stepLines{w: cmd.Root().ErrWriter, terminal: isTerminal(cmd.Root().ErrWriter)}
			if err := selection.MakeRepo(ctx, output, version, steps); err != nil {
				return err
			}
			var printed strings.Builder
			for _, need := range selection.AddedByNeeds() {
				fmt.Fprintln(&printed, need)
			}
			fmt.Fprintf(&printed, "Made %s from the starter %s.\n", output, selection.StarterName())
			if asked {
				fmt.Fprintf(&printed, "To make the same repo again, run:\n  feature-compose new %s\n", selection.Arguments(output))
			}
			_, err = io.WriteString(cmd.Root().Writer, printed.String())
			return err
		},
	}
}

func selected(c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (catalog.Selection, error) {
	starter, err := c.Starter(starterName)
	if err != nil {
		return catalog.Selection{}, err
	}
	return starter.Select(name, given, givenByFeature, catalog.Answers{})
}

func ask(ctx context.Context, cmd *cli.Command, c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (catalog.Selection, error) {
	model, err := form.New(c, starterName, name, given, givenByFeature)
	if err != nil {
		return catalog.Selection{}, err
	}
	final, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(cmd.Root().Reader), tea.WithOutput(cmd.Root().Writer), tea.WithAltScreen()).Run()
	if err != nil {
		return catalog.Selection{}, err
	}
	return final.(form.Model).Confirmed()
}

func givenStarterValues(flags []string) (map[string]string, error) {
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

func givenFeatureValues(flags []string) (map[string]map[string]string, error) {
	given := make(map[string]map[string]string, len(flags))
	for _, flag := range flags {
		featureName, pairs, hasOptions := strings.Cut(flag, ":")
		values := given[featureName]
		if values == nil {
			values = map[string]string{}
		}
		if hasOptions {
			for _, pair := range strings.Split(pairs, ",") {
				optionName, value, ok := strings.Cut(pair, "=")
				if !ok || optionName == "" {
					return nil, fmt.Errorf("the --feature value %q has the pair %q, which is not <option>=<value>: give the flag as <name>:<option>=<value>", flag, pair)
				}
				if earlier, ok := values[optionName]; ok && earlier != value {
					return nil, fmt.Errorf("the option %s of the feature %s has two values: %q and %q", optionName, featureName, earlier, value)
				}
				values[optionName] = value
			}
		}
		given[featureName] = values
	}
	return given, nil
}
