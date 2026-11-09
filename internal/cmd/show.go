package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/browser"
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
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
			starterName := cmd.Args().First()
			shown := definitions
			if starterName != "" {
				shown, err = definitions.ForStarter(starterName)
				if err != nil {
					return err
				}
			}
			if cmd.String("output") == "" && isTerminal(cmd.Root().Reader) && isTerminal(cmd.Root().Writer) {
				return browse(ctx, cmd, definitions, starterName)
			}
			encoder := json.NewEncoder(cmd.Root().Writer)
			encoder.SetIndent("", "  ")
			return encoder.Encode(shown)
		},
	}
}

func browse(ctx context.Context, cmd *cli.Command, definitions catalog.Definitions, starterName string) error {
	model := browser.New(definitions)
	if starterName != "" {
		var err error
		model, err = browser.NewAtStarter(definitions, starterName)
		if err != nil {
			return err
		}
	}
	_, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(cmd.Root().Reader), tea.WithOutput(cmd.Root().Writer), tea.WithAltScreen()).Run()
	return err
}
